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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/schema"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

func NewCmdList(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all skaffold builders currently available locally.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
	return cmd
}

func runList() error {
	path := filepath.Join(os.Getenv("HOME"), ".skaffold/skaffold-builders-manifests/plugins")
	if _, err := os.Stat(path); err != nil {
		return errors.Errorf("%s does not exist, please run 'skaffold update' first", path)
	}
	manifests, err := getManifests()
	if err != nil {
		return err
	}
	fmt.Printf("List of available builders: \n")
	for _, m := range manifests {
		fmt.Println(m.Name)
	}
	return nil
}

func getManifests() ([]schema.PluginManifest, error) {
	path := filepath.Join(os.Getenv("HOME"), ".skaffold/skaffold-builders-manifests/plugins")
	if _, err := os.Stat(path); err != nil {
		return nil, errors.Errorf("%s does not exist, please run 'skaffold update' first", path)
	}
	var manifests []schema.PluginManifest
	err := filepath.Walk(path, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "reading %s", path)
		}
		var m schema.PluginManifest
		if err := yaml.Unmarshal(contents, &m); err != nil {
			return errors.Wrapf(err, "unmarshalling %s", path)
		}
		manifests = append(manifests, m)
		return nil
	})
	return manifests, err
}
