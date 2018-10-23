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
	"io"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/pipeline"

	"github.com/spf13/cobra"
)

// NewCmdPipeline returns basic info about the `skaffold pipeline` command
func NewCmdPipeline(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "A set of commands for setting up a simple CI/CD pipeline by Cloud provider.",
	}

	cmd.AddCommand(pipeline.NewCmdGCB(out))
	return cmd
}
