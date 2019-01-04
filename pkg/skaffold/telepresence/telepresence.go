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

package telepresence

import (
	"fmt"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

type Telepresence struct {
	cancel chan bool
	deployments []string
}

func New(deployments []string) *Telepresence {
	return &Telepresence{
		cancel: make(chan bool),
		deployments: deployments,
	}
}

func (t *Telepresence) Start(artifacts []build.Artifact) error {
	for _, d := range t.deployments {
		images, err := kubernetes.ParseKubernetesYaml(d)
		if err != nil {
			fmt.Printf("Unable to get images in deployment %s, skipping telepresence deployment: %v \n", d, err)
		}
		// TODO: Figure out what to do with more than one image
		if len(images) == 0 {
			fmt.Printf("Couldn't find any images in deployment %s, skipping", d)
		}

		var image string
		for _, a := range artifacts {
			if images[0] == a.ImageName {
				image = fmt.Sprintf("%s:%s", a.ImageName, a.Tag)
			}
		}

		cmd := exec.Command("telepresence", "--swap-deployment", d, "--logfile", "~/skaffold/log", "--docker-run", "--rm", "-it", image)

		go func(){
			fmt.Printf("Swapping deployment telepresence deployment and image %s", image)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			go func() {
				<-t.cancel
				if err := cmd.Process.Kill(); err != nil {
					logrus.Errorf("failed to kill process: %v", err)
				}
			}
			if err := cmd.Start(); err != nil {
				return errors.Wrap(err, "swapping telepresence deployment")
			}
		}
	}
	return nil
}

func (t *Telepresence) Kill() error {
	t.cancel <- true
	return nil
}
