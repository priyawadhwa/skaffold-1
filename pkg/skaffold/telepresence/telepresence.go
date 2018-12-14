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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Item struct {
	Telepresence *latest.Telepresence
}

func NewItem(a *latest.Artifact, e watch.Events, builds []build.Artifact, t []latest.Telepresence) (*Item, error) {
		// If there are no changes, short circuit and don't change anything
		if !e.HasChanged() {
			return nil, nil
		}
}


func something() error {
	if len(k.KubectlDeploy.Telepresence) == 0 {
		return nil
	}
	t := k.KubectlDeploy.Telepresence[0]
	var image string
	for _, b := range builds {
		if b.ImageName == t.Image {
			image = b.Tag
		}
	}
	cmd := exec.Command("telepresence", "--swap-deployment", t.Name, "--logfile", "~/.skaffold/telepresence.log", "--docker-run", image)
	cmd.Stdin = os.Stdin
	k.cmd = cmd
	go func() {
		fmt.Println("Starting telepresence...")
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(string(output))
			fmt.Println("telepresence error")
			return
		}
	}()
	return nil
}
}