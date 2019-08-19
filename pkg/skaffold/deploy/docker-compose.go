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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// DockerComposeDeployer deploys workflows using DockerCompose CLI.
type DockerComposeDeployer struct {
	kubectl            kubectl.CLI
	defaultRepo        string
	insecureRegistries map[string]bool

	cancel context.CancelFunc
}

func NewDockerComposeDeployer(runCtx *runcontext.RunContext) *DockerComposeDeployer {
	return &DockerComposeDeployer{
		kubectl: kubectl.CLI{
			ForceDeploy: runCtx.Opts.ForceDeploy(),
		},
		defaultRepo:        runCtx.DefaultRepo,
		insecureRegistries: runCtx.InsecureRegistries,
	}
}

// Labels returns the labels specific to DockerCompose.
func (k *DockerComposeDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "DockerCompose",
	}
}

// Deploy runs `kubectl apply` on the manifest generated by DockerCompose.
func (k *DockerComposeDeployer) Deploy(parentCtx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) *Result {
	// First, get docker-compose contents
	ml := kubectl.ManifestList{}
	contents, err := ioutil.ReadFile("docker-compose.yaml")
	if err != nil {
		return NewDeployErrorResult(err)
	}
	ml = append(ml, contents)
	// now, replace images
	ml, err = ml.ReplaceImages(builds, k.defaultRepo)
	if err != nil {
		return NewDeployErrorResult(err)
	}
	if k.cancel != nil {
		k.cancel()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	k.cancel = cancel

	cmd := exec.CommandContext(ctx, "docker-compose", "-f", "-", "up", "--no-build", "-d")
	cmd.Stdin = bytes.NewBuffer(ml[0])
	if _, err := util.RunCmdOut(cmd); err != nil {
		return NewDeployErrorResult(errors.Wrapf(err, "running docker-compose up: %v", err))
	}

	cmd = exec.CommandContext(ctx, "docker-compose", "logs", "-f")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return NewDeployErrorResult(errors.Wrap(err, "getting stdout pipe"))
	}
	scanner := bufio.NewScanner(stdoutPipe)
	go func() {
		for scanner.Scan() {
			fmt.Printf("%s\n", scanner.Text())
		}
	}()

	if err := cmd.Start(); err != nil {
		return NewDeployErrorResult(errors.Wrap(err, "starting docker-compose up"))
	}
	go cmd.Wait()
	return NewDeploySuccessResult(nil)
}

func (k *DockerComposeDeployer) Dependencies() ([]string, error) {
	return []string{"docker-compose.yaml"}, nil
}

func (k *DockerComposeDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	if k.cancel != nil {
		k.cancel()
	}
	cmd := exec.CommandContext(ctx, "docker-compose", "down")
	if output, err := util.RunCmdOut(cmd); err != nil {
		return errors.Wrapf(err, "running docker-compose down: %v \n %s", err, string(output))
	}
	return nil
}
