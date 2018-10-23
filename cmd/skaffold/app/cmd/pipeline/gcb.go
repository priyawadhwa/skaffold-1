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
	"fmt"
	"io"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdGCB represents the skaffold pipeline gcb subcommand
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
				logrus.Error(err)
				return
			}

		},
		Args: cobra.NoArgs,
	}
	return cmd
}

func execute() error {
	// TODO (priyawadhwa@): Fill this in
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
