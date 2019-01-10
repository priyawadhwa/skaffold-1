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

package local

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	cfg *latest.LocalBuild

	LocalDocker  docker.LocalDaemon
	LocalCluster bool
	PushImages   bool
	KubeContext  string

	AlreadyTagged map[string]string
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(cfg *latest.LocalBuild, kubeContext string) (*Builder, error) {
	LocalDocker, err := docker.NewAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "getting docker client")
	}

	localCluster := kubeContext == constants.DefaultMinikubeContext || kubeContext == constants.DefaultDockerForDesktopContext
	var PushImages bool
	if cfg.Push == nil {
		PushImages = !localCluster
		logrus.Debugf("push value not present, defaulting to %t because localCluster is %t", PushImages, localCluster)
	} else {
		PushImages = *cfg.Push
	}

	return &Builder{
		cfg:          cfg,
		KubeContext:  kubeContext,
		LocalDocker:  LocalDocker,
		LocalCluster: localCluster,
		PushImages:   PushImages,
	}, nil
}

// Labels are labels specific to local builder.
func (b *Builder) Labels() map[string]string {
	labels := map[string]string{
		constants.Labels.Builder: "local",
	}

	v, err := b.LocalDocker.ServerVersion(context.Background())
	if err == nil {
		labels[constants.Labels.DockerAPIVersion] = fmt.Sprintf("%v", v.APIVersion)
	}

	return labels
}
