/*
Copyright 2018 The Skaffold Authors

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

package build

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// NewPreBuiltImagesBuilder returns an new instance a Builder that assumes images are
// already built with given fully qualified names.
func NewPreBuiltImagesBuilder(images []string) Builder {
	return &prebuiltImagesBuilder{
		images: images,
	}
}

type prebuiltImagesBuilder struct {
	images []string
}

// Labels are labels applied to deployed resources.
func (b *prebuiltImagesBuilder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "pre-built",
	}
}

func (b *prebuiltImagesBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, env latest.ExecutionEnvironment) ([]Artifact, error) {
	tags := make(map[string]string)

	for _, tag := range b.images {
		parsed, err := docker.ParseReference(tag)
		if err != nil {
			return nil, err
		}

		tags[parsed.BaseName] = tag
	}

	var builds []Artifact

	for _, artifact := range artifacts {
		tag, present := tags[artifact.ImageName]
		if !present {
			return nil, errors.Errorf("unable to find image tag for %s", artifact.ImageName)
		}
		delete(tags, artifact.ImageName)

		builds = append(builds, Artifact{
			ImageName: artifact.ImageName,
			Tag:       tag,
		})
	}

	for image, tag := range tags {
		builds = append(builds, Artifact{
			ImageName: image,
			Tag:       tag,
		})
	}

	return builds, nil
}
