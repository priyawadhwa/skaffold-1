/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"context"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// For testing
	buildComplete = event.BuildComplete
)

func (c *cache) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildAndTest BuildAndTestFn) ([]build.Artifact, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()

	color.Default.Fprintln(out, "Checking cache...")

	lookup := make(chan []cacheDetails)
	go func() { lookup <- c.lookupArtifacts(ctx, tags, artifacts) }()

	var results []cacheDetails
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case results = <-lookup:
	}

	hashByName := make(map[string]string)
	var needToBuild []*latest.Artifact
	var alreadyBuilt []build.Artifact
	for i, artifact := range artifacts {
		color.Default.Fprintf(out, " - %s: ", artifact.ImageName)

		result := results[i]
		switch result := result.(type) {
		case failed:
			logrus.Warnf("error checking cache, caching may not work as expected: %v", result.err)
			color.Yellow.Fprintln(out, "Error checking cache. Rebuilding.")
			needToBuild = append(needToBuild, artifact)
			continue

		case needsBuilding:
			color.Yellow.Fprintln(out, "Not found. Building")
			hashByName[artifact.ImageName] = result.Hash()
			needToBuild = append(needToBuild, artifact)
			continue

		case needsTagging:
			color.Green.Fprintln(out, "Found. Tagging")
			if err := result.Tag(ctx, c); err != nil {
				return nil, errors.Wrap(err, "tagging image")
			}

		case needsPushing:
			color.Green.Fprintln(out, "Found. Pushing")
			if err := result.Push(ctx, out, c); err != nil {
				return nil, errors.Wrap(err, "pushing image")
			}

		default:
			if c.imagesAreLocal {
				color.Green.Fprintln(out, "Found Locally")
			} else {
				color.Green.Fprintln(out, "Found Remotely")
			}
		}

		// Image is already built
		buildComplete(artifact.ImageName)
		entry := c.artifactCache[result.Hash()]
		var uniqueTag string
		if c.imagesAreLocal {
			var err error
			uniqueTag, err = c.client.TagWithImageID(ctx, artifact.ImageName, entry.ID)
			if err != nil {
				return nil, err
			}
		} else {
			uniqueTag = tags[artifact.ImageName] + "@" + entry.Digest
		}

		alreadyBuilt = append(alreadyBuilt, build.Artifact{
			ImageName: artifact.ImageName,
			Tag:       uniqueTag,
		})
	}

	color.Default.Fprintln(out, "Cache check complete in", time.Since(start))

	bRes, err := buildAndTest(ctx, out, tags, needToBuild)
	if err != nil {
		return nil, errors.Wrap(err, "build failed")
	}

	if err := c.addArtifacts(ctx, bRes, hashByName); err != nil {
		logrus.Warnf("error adding artifacts to cache; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	if err := saveArtifactCache(c.cacheFile, c.artifactCache); err != nil {
		logrus.Warnf("error saving cache file; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	return append(bRes, alreadyBuilt...), err
}

func (c *cache) addArtifacts(ctx context.Context, bRes []build.Artifact, hashByName map[string]string) error {
	for _, a := range bRes {
		entry := ImageDetails{}

		if !c.imagesAreLocal {
			ref, err := docker.ParseReference(a.Tag)
			if err != nil {
				return errors.Wrapf(err, "parsing reference %s", a.Tag)
			}

			entry.Digest = ref.Digest
		}

		imageID, err := c.client.ImageID(ctx, a.Tag)
		if err != nil {
			return err
		}

		if imageID != "" {
			entry.ID = imageID
		}

		c.artifactCache[hashByName[a.ImageName]] = entry
	}

	return nil
}
