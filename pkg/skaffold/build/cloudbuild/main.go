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
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
)

func main() {
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"cloudbuild": &shared.BuilderPlugin{Impl: newBuilder()},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         pluginMap,
	})
}

type CloudbuildBuilderPlugin struct {
	Impl build.Builder
}

func (b *CloudbuildBuilderPlugin) Labels() map[string]string {
	return b.Impl.Labels()
}

func (b *CloudbuildBuilderPlugin) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return b.Impl.Build(ctx, os.Stderr, tagger, artifacts)
}

func newBuilder() build.Builder {
	return &CloudbuildBuilderPlugin{
		Impl: gcb.NewBuilder(&latest.GoogleCloudBuild{
			ProjectID:   "priya-wadhwa",
			DockerImage: constants.DefaultCloudBuildDockerImage,
		}),
	}
}
