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

package sync

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Syncer interface {
	Sync(context.Context, *Item) error
}

type Item struct {
	Image  string
	Copy   map[string]string
	Delete map[string]string
}

func NewItem(a *latest.Artifact, e watch.Events, builds []build.Artifact) (*Item, error) {
	// If there are no changes, short circuit and don't sync anything
	if !e.HasChanged() || len(a.Sync) == 0 {
		return nil, nil
	}

	toCopy, err := intersect(a.Workspace, a.Sync, append(e.Added, e.Modified...))
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and added, modified files")
	}

	toDelete, err := intersect(a.Workspace, a.Sync, e.Deleted)
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and deleted files")
	}

	// Something went wrong, don't sync, rebuild.
	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	return &Item{
		Image:  tag,
		Copy:   toCopy,
		Delete: toDelete,
	}, nil
}

func latestTag(image string, builds []build.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

func intersect(context string, syncMap map[string]string, files []string) (map[string]string, error) {
	ret := map[string]string{}
	for _, f := range files {
		relPath, err := filepath.Rel(context, f)
		if err != nil {
			return nil, errors.Wrapf(err, "changed file %s can't be found relative to context %s", f, context)
		}
		for p, dst := range syncMap {
			match, err := filepath.Match(p, relPath)
			if err != nil {
				return nil, errors.Wrapf(err, "pattern error for %s", relPath)
			}
			if !match {
				return nil, nil
			}
			// If the source has special match characters,
			// the destination must be a directory
			// The path package must be used here, since the destination is always
			// a linux filesystem.
			if util.HasMeta(p) {
				dst = path.Join(dst, filepath.Base(relPath))
			}
			ret[f] = dst
		}
	}
	return ret, nil
}

func Perform(ctx context.Context, image string, files map[string]string, cmdFn func(context.Context, v1.Pod, v1.Container, string, string) *exec.Cmd) error {
	if len(files) == 0 {
		return nil
	}

	client, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting k8s client")
	}

	pods, err := client.CoreV1().Pods("").List(meta_v1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "getting pods")
	}

	synced := map[string]bool{}

	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			if c.Image != image {
				continue
			}

			for src, dst := range files {
				cmd := cmdFn(ctx, p, c, src, dst)
				if err := util.RunCmd(cmd); err != nil {
					return err
				}

				synced[src] = true
			}
		}
	}

	if len(synced) != len(files) {
		return errors.New("couldn't sync all the files")
	}

	return nil
}
