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

package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/krew/pkg/gitutil"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// search for new skaffold builders
func NewCmdUpdate(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update local repo to match repo of skaffold builders on GitHub",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
	return cmd
}

func runUpdate() error {
	// Copy repo to ~/.skaffold/plugins/skaffold-builders-manifests
	path := filepath.Join(os.Getenv("HOME"), ".skaffold/skaffold-builders-manifests")
	if err := gitutil.EnsureUpdated(constants.BuilderPluginManifestRepo, path); err != nil {
		return errors.Wrap(err, "failed to update the local index")
	}
	fmt.Fprintln(os.Stderr, "Updated the local copy of builder plugins index.")
	return nil
}
