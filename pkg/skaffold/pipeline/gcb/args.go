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

package gcb

import (
	"fmt"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

const (
	other = "Other"
)

type Args struct {
	Project      string
	Cluster      string
	Zone         string
	YamlFilepath string
}

func NewArgs() (*Args, error) {
	fmt.Println("Let's get started setting up a simple CI/CD pipeline for your repo!\nTo do this we need to collect some basic information.")
	// First, get the project
	cp, err := currentProject()
	if err != nil {
		return nil, errors.Wrap(err, "getting current  project")
	}
	cp, err = userSelectedList("Google Cloud Platform project", []string{cp, other})
	if err != nil {
		return nil, err
	}
	// Next, get the current context:
	ctx, err := context.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current context")
	}
	ctx, err = userSelectedList(fmt.Sprintf("Kubernetes cluster within project %s", cp), []string{ctx, other, "Create new cluster"})
	if err != nil {
		return nil, err
	}
	return &Args{
		Project: cp,
		Cluster: ctx,
	}, nil
}

func currentProject() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	bytes, err := util.RunCmdOut(cmd)
	return string(bytes), err
}
