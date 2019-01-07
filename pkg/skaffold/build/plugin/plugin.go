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
	"log"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/hashicorp/go-plugin"
)

func NewPluginBuilder(pluginName string) build.Builder {
	// We're a host. Start by launching the plugin process.
	log.SetOutput(os.Stdout)

	client := plugin.NewClient(&plugin.ClientConfig{
		Stderr:          os.Stderr,
		SyncStderr:      os.Stderr,
		SyncStdout:      os.Stdout,
		Managed:         true,
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(pluginName),
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	return &PluginBuilder{
		Builder: raw.(build.Builder),
	}
}

type PluginBuilder struct {
	build.Builder
}

// Labels are labels applied to deployed resources.
func (b *PluginBuilder) Labels() map[string]string {
	return b.Builder.Labels()
}

func (b *PluginBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return b.Builder.Build(ctx, out, tagger, artifacts)
}
