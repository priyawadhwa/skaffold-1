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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateMavenArgs(t *testing.T) {
	var testCases = []struct {
		in  latest.JibMavenArtifact
		out []string
	}{
		{latest.JibMavenArtifact{}, []string{"--non-recursive", "prepare-package", "jib:goal", "-Dimage=image"}},
		{latest.JibMavenArtifact{Profile: "profile"}, []string{"--non-recursive", "prepare-package", "jib:goal", "-Dimage=image", "--activate-profiles", "profile"}},
		{latest.JibMavenArtifact{Module: "module"}, []string{"--projects", "module", "--also-make", "package", "-Dimage=image"}},
		{latest.JibMavenArtifact{Module: "module", Profile: "profile"}, []string{"--projects", "module", "--also-make", "package", "-Dimage=image", "--activate-profiles", "profile"}},
	}

	for _, tt := range testCases {
		args := generateMavenArgs("goal", "image", &tt.in)

		testutil.CheckDeepEqual(t, tt.out, args)
	}
}

func TestMavenVerifyJibPackageGoal(t *testing.T) {
	var testCases = []struct {
		requiredGoal string
		mavenOutput  string
		shouldError  bool
	}{
		{"xxx", "", true},   // no goals should fail
		{"xxx", "\n", true}, // no goals should fail; newline stripped
		{"dockerBuild", "dockerBuild", false},
		{"dockerBuild", "dockerBuild\n", false}, // newline stripped
		{"dockerBuild", "build\n", true},
		{"dockerBuild", "build\ndockerBuild\n", true},
	}

	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.SkipWrapperCheck = true

	for _, tt := range testCases {
		util.DefaultExecCommand = testutil.NewFakeCmdOut("mvn --quiet --projects module jib:_skaffold-package-goals", tt.mavenOutput, nil)

		err := verifyJibPackageGoal(context.Background(), tt.requiredGoal, ".", &latest.JibMavenArtifact{Module: "module"})
		if hasError := err != nil; tt.shouldError != hasError {
			t.Error("Unexpected return result")
		}
	}

}

func TestGenerateGradleArgs(t *testing.T) {
	var testCases = []struct {
		in  latest.JibGradleArtifact
		out []string
	}{
		{latest.JibGradleArtifact{}, []string{":task", "--image=image"}},
		{latest.JibGradleArtifact{Project: "project"}, []string{":project:task", "--image=image"}},
	}

	for _, tt := range testCases {
		command := generateGradleArgs("task", "image", &tt.in)

		testutil.CheckDeepEqual(t, tt.out, command)
	}
}

func TestGenerateJibImageRef(t *testing.T) {
	var testCases = []struct {
		workspace string
		project   string
		out       string
	}{
		{"simple", "", "jibsimple"},
		{"simple", "project", "jibsimple_project"},
		{".", "project", "jib__d8c7cbe8892fe8442b7f6ef42026769ee6a01e67"},
		{"complex/workspace", "project", "jib__965ec099f720d3ccc9c038c21ea4a598c9632883"},
	}

	for _, tt := range testCases {
		computed := generateJibImageRef(tt.workspace, tt.project)

		testutil.CheckDeepEqual(t, tt.out, computed)
	}
}
