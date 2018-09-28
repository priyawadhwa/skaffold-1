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

package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

type GithubRepo struct {
	Organization string
	Repo         string
}

// Organizations returns a list of orgs this user has access to
func Organizations(user string) ([]string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)
	orgs, _, err := client.Organizations.List(ctx, user, nil)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting organizations under github user %s", user))
	}
	var organizations []string
	for _, o := range orgs {
		if o.Name != nil {
			organizations = append(organizations, *o.Name)
		}
	}
	return organizations, nil
}

// Repos returns a list of repos the user has access to
func Repos(user string) ([]string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)
	githubRepos, _, err := client.Repositories.List(ctx, user, nil)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting repositories under github user %s", user))
	}
	var repos []string
	for _, gr := range githubRepos {
		if gr.Name != nil {
			repos = append(repos, *gr.Name)
		}
	}
	return repos, nil
}

func currentGithubRepo() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "getting cwd")
	}
	ok := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Set up CI/CD for the current repo: %s", filepath.Base(cwd)),
	}
	survey.AskOne(prompt, &ok, nil)
	if !ok {
		return "", fmt.Errorf("please set up CI/CD solution desired GitHub repo")
	}
	return cwd, err
}
