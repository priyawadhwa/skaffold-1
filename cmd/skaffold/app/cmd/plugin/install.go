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
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/schema"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCmdInstall(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a skaffold builder locally.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(args[0])
		},
	}
	return cmd
}

func runInstall(builder string) error {
	fmt.Printf("Installing %s... \n", builder)
	manifests, err := getManifests()
	if err != nil {
		return errors.Wrap(err, "getting builder manifests")
	}
	var manifest *schema.PluginManifest
	for _, m := range manifests {
		if m.Name == builder {
			manifest = &m
		}
	}
	if manifest == nil {
		return errors.Errorf("%s is not a valid local skaffold builder, please try 'skaffold update' to update local repo of builders", builder)
	}

	if err := installManifest(manifest); err != nil {
		return err
	}

	fmt.Printf("Successfully installed %s. \n", builder)
	return nil
}

func installManifest(m *schema.PluginManifest) error {
	var uri string
	for _, p := range m.Platforms {
		if p.OS == runtime.GOOS {
			uri = p.URI
		}
	}

	if uri == "" {
		return errors.Errorf("%s has no supported version for %s", m.Name, runtime.GOOS)
	}

	return downloadBinary(m.Name, uri)
}

func downloadBinary(name, uri string) error {
	path := filepath.Join(os.Getenv("HOME"), ".skaffold/builders")
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Errorf("creating %s", path)
	}
	binaryPath := filepath.Join(path, name)
	resp, err := http.Get(uri)
	if err != nil {
		return errors.Wrapf(err, "getting %s", uri)
	}
	if err := removeBinary(binaryPath); err != nil {
		return errors.Wrapf(err, "removing %s", binaryPath)
	}

	f, err := os.Create(binaryPath)
	if err != nil {
		return errors.Wrapf(err, "opening %s", binaryPath)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return errors.Wrapf(err, "copying binary to %s", binaryPath)
	}
	return os.Chmod(binaryPath, 0777)
}

func removeBinary(path string) error {
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}
