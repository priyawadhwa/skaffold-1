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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, artifact *latest.Artifact) (string, error) {
	if b.PushImages {
		return b.buildJibGradleToRegistry(ctx, out, workspace, artifact)
	}
	return b.buildJibGradleToDocker(ctx, out, workspace, artifact.JibGradleArtifact)
}

func (b *Builder) buildJibGradleToDocker(ctx context.Context, out io.Writer, workspace string, artifact *latest.JibGradleArtifact) (string, error) {
	skaffoldImage := generateJibImageRef(workspace, artifact.Project)
	args := generateGradleArgs("jibDockerBuild", skaffoldImage, artifact)

	if err := runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return b.LocalDocker.ImageID(ctx, skaffoldImage)
}

func (b *Builder) buildJibGradleToRegistry(ctx context.Context, out io.Writer, workspace string, artifact *latest.Artifact) (string, error) {
	initialTag := util.RandomID()
	skaffoldImage := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	args := generateGradleArgs("jib", skaffoldImage, artifact.JibGradleArtifact)

	if err := runGradleCommand(ctx, out, workspace, args); err != nil {
		return "", err
	}

	return docker.RemoteDigest(skaffoldImage)
}

// generateGradleArgs generates the arguments to Gradle for building the project as an image called `skaffoldImage`.
func generateGradleArgs(task string, imageName string, artifact *latest.JibGradleArtifact) []string {
	var command string
	if artifact.Project == "" {
		command = ":" + task
	} else {
		// multi-module
		command = fmt.Sprintf(":%s:%s", artifact.Project, task)
	}

	return []string{command, "--image=" + imageName}
}

func runGradleCommand(ctx context.Context, out io.Writer, workspace string, args []string) error {
	cmd := jib.GradleCommand.CreateCommand(ctx, workspace, args)
	cmd.Stdout = out
	cmd.Stderr = out

	logrus.Infof("Building %s: %s, %v", workspace, cmd.Path, cmd.Args)
	if err := util.RunCmd(cmd); err != nil {
		return errors.Wrap(err, "gradle build failed")
	}

	return nil
}
