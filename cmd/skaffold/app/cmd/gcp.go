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

package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	cloudbuild "google.golang.org/api/cloudbuild/v1/generated"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func NewCmdGCP(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Set up a CI/CD pipeline with GitHub and Google Cloud Build",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := meetsRequirements(); err != nil {
				logrus.Error(err)
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := requestGithubUsername(); err != nil {
				fmt.Println(err)
				return
			}
			if err := installGCBGithubApp(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("DONE")
		},
		Args: cobra.NoArgs,
	}
	return cmd
}

func meetsRequirements() error {
	requiredTools := []string{"gcloud"}
	for _, tool := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("you must have %s installed and on your PATH", tool)
		}
	}
	return nil
}

func retrieveCluster() error {
	// TODO: Make sure the cluster is correct, otherwise allow user to select alternative clusters from kubeconfig

	// TODO: Give option of creating a new cluster
	return nil
}

func checkIfGCBAppInstalled(org, project string) error {
	return nil
}

func installGCBGithubApp() error {
	// Open browser window to go through flow
	fmt.Println("Please install the Google Cloud Build GitHub app on your repo\n Note: This will require giving Google Cloud Platform permissions to your account.")
	return open.Run("https://github.com/apps/google-cloud-build")
}

func getAvailableGithubRepos() error {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		return errors.Wrap(err, "getting google client")
	}
	cbclient, err := cloudbuild.New(client)
	if err != nil {
		return errors.Wrap(err, "getting builder")
	}
	cbclient.UserAgent = version.UserAgent()
	return nil
}

func createBuildTrigger() error {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		return errors.Wrap(err, "getting google client")
	}
	cbclient, err := cloudbuild.New(client)
	if err != nil {
		return errors.Wrap(err, "getting builder")
	}
	cbclient.UserAgent = version.UserAgent()

	bt := &cloudbuild.BuildTrigger{
		Github: &cloudbuild.GitHubEventsConfig{},
	}
	fmt.Println(bt)
	return nil
}

// requestGithubUsername requests the user to input their GitHub username
func requestGithubUsername() (string, error) {
	user := ""
	correct := false

	for {
		if correct {
			break
		}
		input := &survey.Input{
			Message: "Please enter your GitHub username:",
		}
		if err := survey.AskOne(input, &user, nil); err != nil {
			return "", err
		}
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Is %s correct?", user),
		}
		if err := survey.AskOne(confirm, &correct, nil); err != nil {
			return "", err
		}
	}

	return user, nil
}

// func promptUserForGithubRepo() (string, error) {
// 	var selectedDockerfile string
// 	options := append(dockerfiles, NoDockerfile)
// 	prompt := &survey.Select{
// 		Message:  fmt.Sprintf("Choose the dockerfile to build image %s", image),
// 		Options:  options,
// 		PageSize: 15,
// 	}
// 	survey.AskOne(prompt, &selectedDockerfile, nil)
// 	return nil, dockerfilePair{
// 		Dockerfile: selectedDockerfile,
// 		ImageName:  image,
// 	}
// }
