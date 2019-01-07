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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/plugin"
	"github.com/spf13/cobra"
)

func NewCmdPlugin(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "A set of commands for interacting with skaffold builder plugins.",
	}

	cmd.AddCommand(plugin.NewCmdUpdate(out))
	cmd.AddCommand(plugin.NewCmdList(out))
	cmd.AddCommand(plugin.NewCmdInstall(out))
	return cmd
}
