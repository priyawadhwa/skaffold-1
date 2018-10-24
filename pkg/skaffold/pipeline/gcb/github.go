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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// GithubProject contains necessary information about the github project
// we are trying to set up CI/CD for
type GithubProject struct {
	Organization string
	Repo         string
}

// NewGithubProject returns information about the current github project
func NewGithubProject() (*GithubProject, error) {
	repo, err := retrieveRepo()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving repo")
	}
	org, err := retrieveOrg(repo)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving org")
	}
	return &GithubProject{
		Repo:         repo,
		Organization: org,
	}, nil
}

func retrieveRepo() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "getting cwd")
	}
	return filepath.Base(wd), nil
}

func retrieveOrg(repo string) (string, error) {
	orgs, err := listOrgs()
	if err != nil {
		return "", err
	}

	org := ""
	prompt := &survey.Select{
		Message: fmt.Sprintf("Please select an associated GitHub organization for the repository %s", repo),
		Options: orgs,
	}
	if err := survey.AskOne(prompt, &org, nil); err != nil {
		return "", err
	}
	return org, nil
}

func listOrgs() ([]string, error) {
	cmd := exec.Command("git", "config", "--local", "--get-regexp", "remote.*.url")
	data, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, err
	}
	return parseRemotes(data), nil
}

func parseRemotes(data []byte) []string {
	remotes := strings.Split(string(data), "\n")
	var orgs []string
	for _, r := range remotes {
		// Parse out the org from the remote, which is the form "remote.origin.url git@github.com:priyawadhwa/runtimes-common.git"
		// we want "priyawadhwa"
		split := strings.Split(r, ":")
		if len(split) == 1 {
			continue
		}
		org := strings.Split(split[1], "/")[0]
		orgs = append(orgs, org)
	}
	return orgs
}
