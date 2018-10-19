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

package jib

import (
	"context"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var MavenCommand = util.CommandWrapper{Executable: "mvn", Wrapper: "mvnw"}

// GetDependenciesMaven finds the source dependencies for the given jib-maven artifact.
// All paths are absolute.
func GetDependenciesMaven(ctx context.Context, workspace string, a *latest.JibMavenArtifact) ([]string, error) {
	deps, err := getDependencies(getCommandMaven(ctx, workspace, a))
	if err != nil {
		return nil, errors.Wrapf(err, "getting jibMaven dependencies")
	}
	logrus.Debugf("Found dependencies for jibMaven artifact: %v", deps)
	return deps, nil
}

func getCommandMaven(ctx context.Context, workspace string, a *latest.JibMavenArtifact) *exec.Cmd {
	args := []string{"--quiet"}
	if a.Module == "" {
		// single-module project
		args = append(args, "--non-recursive")
	} else {
		// multi-module project
		args = append(args, "--projects", a.Module, "--also-make")
	}
	args = append(args, "jib:_skaffold-files")
	if a.Profile != "" {
		args = append(args, "--activate-profiles", a.Profile)
	}

	return MavenCommand.CreateCommand(ctx, workspace, args)
}
