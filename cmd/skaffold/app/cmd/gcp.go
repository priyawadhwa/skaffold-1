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
	"golang.org/x/oauth2/google"
	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/spf13/cobra"
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

func installGCBGithubApp() error {
	// Open browser window to go through flow
	return nil
}

func getAvailableGithubRepose() error {
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
	cbclient.Projects.Builds.
	return nil
}

type Github struct {
	Project string
}
