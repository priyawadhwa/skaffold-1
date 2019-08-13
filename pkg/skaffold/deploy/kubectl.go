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

package deploy

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// KubectlDeployer deploys workflows using kubectl CLI.
type KubectlDeployer struct {
	*latest.KubectlDeploy

	originalImages     []build.Artifact
	workingDir         string
	kubectl            deploy.CLI
	defaultRepo        string
	insecureRegistries map[string]bool
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(runCtx *runcontext.RunContext) *KubectlDeployer {
	return &KubectlDeployer{
		KubectlDeploy: runCtx.Cfg.Deploy.KubectlDeploy,
		workingDir:    runCtx.WorkingDir,
		kubectl: deploy.CLI{
			CLI:         kubectl.NewFromRunContext(runCtx),
			Flags:       runCtx.Cfg.Deploy.KubectlDeploy.Flags,
			ForceDeploy: runCtx.Opts.ForceDeploy(),
		},
		defaultRepo:        runCtx.DefaultRepo,
		insecureRegistries: runCtx.InsecureRegistries,
	}
}

func (k *KubectlDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kubectl",
	}
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) error {
	color.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		color.Default.Fprintln(out, err)
	}

	manifests, err := k.readManifests(ctx)
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "reading manifests")
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return errors.Wrap(err, "get remote manifests")
		}

		manifests = append(manifests, manifest)
	}

	if len(k.originalImages) == 0 {
		k.originalImages, err = manifests.GetImages()
		if err != nil {
			return errors.Wrap(err, "get images from manifests")
		}
	}

	logrus.Debugln("manifests", manifests.String())

	if len(manifests) == 0 {
		return nil
	}

	event.DeployInProgress()

	manifests, err = manifests.ReplaceImages(builds, k.defaultRepo)
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "replacing images in manifests")
	}

	manifests, err = manifests.SetLabels(merge(labellers...))
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "setting labels in manifests")
	}

	for _, transform := range manifestTransforms {
		manifests, err = transform(manifests, builds, k.insecureRegistries)
		if err != nil {
			event.DeployFailed(err)
			return errors.Wrap(err, "unable to transform manifests")
		}
	}

	if err := k.kubectl.Apply(ctx, out, manifests); err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "kubectl error")
	}

	event.DeployComplete()
	return nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *KubectlDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := k.readManifests(ctx)
	if err != nil {
		return errors.Wrap(err, "reading manifests")
	}

	// pull remote manifests
	var rm deploy.ManifestList
	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return errors.Wrap(err, "get remote manifests")
		}
		rm = append(rm, manifest)
	}
	upd, err := rm.ReplaceImages(k.originalImages, k.defaultRepo)
	if err != nil {
		return errors.Wrap(err, "replacing with originals")
	}
	if err := k.kubectl.Apply(ctx, out, upd); err != nil {
		return errors.Wrap(err, "apply original")
	}
	if err := k.kubectl.Delete(ctx, out, manifests); err != nil {
		return errors.Wrap(err, "delete")
	}

	return nil
}

func (k *KubectlDeployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}

func (k *KubectlDeployer) manifestFiles(manifests []string) ([]string, error) {
	var nonURLManifests []string
	for _, manifest := range manifests {
		if !util.IsURL(manifest) {
			nonURLManifests = append(nonURLManifests, manifest)
		}
	}

	list, err := util.ExpandPathsGlob(k.workingDir, nonURLManifests)
	if err != nil {
		return nil, errors.Wrap(err, "expanding kubectl manifest paths")
	}

	var filteredManifests []string
	for _, f := range list {
		if !util.IsSupportedKubernetesFormat(f) {
			if !util.StrSliceContains(manifests, f) {
				logrus.Infof("refusing to deploy/delete non {json, yaml} file %s", f)
				logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		filteredManifests = append(filteredManifests, f)
	}

	return filteredManifests, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *KubectlDeployer) readManifests(ctx context.Context) (deploy.ManifestList, error) {
	// Get file manifests
	manifests, err := k.Dependencies()
	if err != nil {
		return nil, errors.Wrap(err, "listing manifests")
	}

	// Append URL manifests
	for _, manifest := range k.KubectlDeploy.Manifests {
		if util.IsURL(manifest) {
			manifests = append(manifests, manifest)
		}
	}

	if len(manifests) == 0 {
		return deploy.ManifestList{}, nil
	}

	return k.kubectl.ReadManifests(ctx, manifests)
}

// readRemoteManifests will try to read manifests from the given kubernetes
// context in the specified namespace and for the specified type
func (k *KubectlDeployer) readRemoteManifest(ctx context.Context, name string) ([]byte, error) {
	var args []string
	ns := ""
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		ns = parts[0]
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := k.kubectl.RunInNamespace(ctx, nil, &manifest, "get", ns, args...)
	if err != nil {
		return nil, errors.Wrap(err, "getting manifest")
	}

	return manifest.Bytes(), nil
}
