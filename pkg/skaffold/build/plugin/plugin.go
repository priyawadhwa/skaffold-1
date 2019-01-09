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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

func NewPluginBuilder(cfg *latest.PluginBuild) build.Builder {
	// We're a host. Start by launching the plugin process.
	log.SetOutput(os.Stdout)

	if err := validate(cfg); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var flags []string
	for _, f := range cfg.Flags {
		flags = append(flags, fmt.Sprintf("-%s=%s", f.Name, f.Value))
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		Stderr:          os.Stderr,
		SyncStderr:      os.Stderr,
		SyncStdout:      os.Stdout,
		Managed:         true,
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(cfg.Name, flags...),
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(cfg.Name)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	return &PluginBuilder{
		Builder: raw.(build.Builder),
	}
}

func validate(cfg *latest.PluginBuild) error {
	m, err := getManifestForBuilder(cfg.Name)
	if err != nil {
		return errors.Wrapf(err, "getting manifest for builder")
	}
	return requiredFlagsExist(m, cfg.Flags)
}

func requiredFlagsExist(m *schema.PluginManifest, flags []latest.Flag) error {
	for _, mf := range m.Flags {
		if mf.Required && !requiredFlagExists(mf.Name, flags) {
			return errors.Errorf("Required flag %s not found in skaffold.yaml", mf.Name)
		}
	}
	return nil
}

func requiredFlagExists(name string, flags []latest.Flag) bool {
	for _, f := range flags {
		if f.Name == name {
			return true
		}
	}
	return false
}

func getManifestForBuilder(builder string) (*schema.PluginManifest, error) {
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
	if err != nil {
		return nil, err
	}
	for _, m := range manifests {
		return &m, nil
	}
	return nil, errors.Errorf("Couldn't get manifest for builder %s", builder)
}

type PluginBuilder struct {
	build.Builder
}

// Labels are labels applied to deployed resources.
func (b *PluginBuilder) Labels() map[string]string {
	return b.Builder.Labels()
}

func (b *PluginBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, env string) ([]build.Artifact, error) {
	return b.Builder.Build(ctx, out, tagger, artifacts, env)
}
