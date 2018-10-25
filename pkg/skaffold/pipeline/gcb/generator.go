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

package gcb

import (
	"encoding/json"
	"io/ioutil"
	"os"

	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

const (
	skaffoldImage  = "gcr.io/k8s-skaffold/skaffold:v0.15.0"
	cloudbuildPath = "cloudbuild.yaml"
)

var template = `


`

func (c *Client) build() *cloudbuild.Build {
	var steps []*cloudbuild.BuildStep

	args := []string{"gcloud", "container", "clusters", "get-credentials", c.Cluster.Name, "--zone", c.Cluster.Zone, "--project", c.GCPPRoject.Name}

	steps = append(steps, &cloudbuild.BuildStep{
		Name: skaffoldImage,
		Args: args,
	})

	args = []string{"skaffold", "run"}

	steps = append(steps, &cloudbuild.BuildStep{
		Name: skaffoldImage,
		Args: args,
	})

	return &cloudbuild.Build{
		Steps: steps,
	}
}

// WriteCloudbuildYaml write the cloudbuild yaml to fp
func (c *Client) WriteCloudbuildYaml() error {
	cb := c.build()
	if _, err := os.Create(cloudbuildPath); err != nil {
		return err
	}
	data, err := json.Marshal(cb)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cloudbuildPath, data, 0644)
}
