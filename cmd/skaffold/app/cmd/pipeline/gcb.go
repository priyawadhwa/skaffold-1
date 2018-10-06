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

package pipeline

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/gcb"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	cloudbuild "google.golang.org/api/cloudbuild/v1/generated"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func NewCmdGCB(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcb",
		Short: "Set up a CI/CD pipeline with GitHub and Google Cloud Build",
		Long: `
		This command assumes you have a Google account and a running GKE cluster associated with a project.
		This command will guide you through setting up the Google Cloud Build GitHub app on your GitHub repo,
		which will provide the authentication necessary for skaffold to set up a GCB build trigger on your project.
		skaffold will generate a cloudbuild.yaml with one step, "skaffold run", which will run everytime a PR is merged on your repo.
		This step will build and deploy all artifacts specified in your skaffold.yaml.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := meetsRequirements(); err != nil {
				logrus.Error(err)
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := execute(); err != nil {
				fmt.Println(err)
				return
			}

		},
		Args: cobra.NoArgs,
	}
	return cmd
}

func execute() error {
	installed, err := isGitHubAppInstalled()
	if err != nil {
		return errors.Wrap(err, "checking if github app is installed")
	}
	if !installed {
		return installGCBGithubApp()
	}
	args, err := gcb.NewArgs()
	if err != nil {
		return err
	}
	fmt.Println(args)
	// Now that the github app is definitely installed, we can go about generating the cloudbuild.yaml

	// Then, create the build trigger

	// Ask the user to commit the generated cloudbuild.yaml

	// Upon merge, we can potentially run `skaffold pipeline gcb logs`?
	return nil
}

func meetsRequirements() error {
	requiredTools := map[string]string{
		"gcloud": "https://cloud.google.com/sdk/install",
	}
	for tool, link := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s must be installed\n Installation instructions can be found here: %s", tool, link)
		}
	}
	return nil
}

func retrieveClusters() error {
	// TODO: Make sure the cluster is correct, otherwise allow user to select alternative clusters from kubeconfig

	// TODO: Give option of creating a new cluster
	return nil
}

func checkIfGCBAppInstalled(org, project string) error {
	return nil
}

func isGitHubAppInstalled() (bool, error) {
	return true, nil
}

func installGCBGithubApp() error {
	// Open browser window to go through flow
	fmt.Println("Please install the Google Cloud Build GitHub app on your repo and rerun `skaffold gcp`\n Note: This will require giving Google Cloud Platform permissions to your account.")
	// TODO: When user hits enter, continue the flow
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
