/*
Copyright 2019 The Skaffold Authors

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

package runner

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

func (r *SkaffoldRunner) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	color.Default.Fprintln(out, "Starting deploy...")
	start := time.Now()
	if config.IsKindCluster(r.runCtx.KubeContext) {
		// With `kind`, docker images have to be loaded with the `kind` CLI.
		if err := r.loadImagesInKindNodes(ctx, out, artifacts); err != nil {
			return errors.Wrapf(err, "loading images into kind nodes")
		}
	}

	deployResult := r.deployer.Deploy(ctx, out, artifacts, r.labellers)
	r.hasDeployed = true
	if err := deployResult.GetError(); err != nil {
		return err
	}
	r.runCtx.UpdateNamespaces(deployResult.Namespaces())
	err := r.performStatusCheck(ctx, out)
	color.Default.Fprintln(out, "Deploy complete in", time.Since(start))
	return err
}

func (r *SkaffoldRunner) performStatusCheck(ctx context.Context, out io.Writer) error {
	// Check if we need to perform deploy status
	if r.runCtx.Opts.StatusCheck {
		color.Default.Fprintln(out, "Waiting for deployments to stabilize")
		err := statusCheck(ctx, r.defaultLabeller, r.runCtx)
		if err != nil {
			color.Default.Fprintln(out, err.Error())
		}
		return err
	}
	return nil
}
