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

package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
)

var (
	projectID string
)

func init() {
	flag.StringVar(&projectID, "projectID", "", "Set the GCP project id")
	flag.Parse()
}

func main() {
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"bazel-skaffold": &shared.BuilderPlugin{Impl: newBuilder()},
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: shared.Handshake,
			Plugins:         pluginMap,
		})
	}()

	<-sigs
	plugin.CleanupClients()
}

type BazelBuilderPlugin struct {
	Impl build.Builder
}

func (b *BazelBuilderPlugin) Labels() map[string]string {
	return b.Impl.Labels()
}

func (b *BazelBuilderPlugin) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, env latest.ExecutionEnvironment) ([]build.Artifact, error) {
	return b.Impl.Build(ctx, os.Stderr, tagger, artifacts, env)
}

func newBuilder() build.Builder {
	return &BazelBuilderPlugin{
		Impl: bazel.NewBuilder(),
	}
}
