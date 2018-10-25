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
	"bytes"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/pkg/errors"
)

const (
	skaffoldImage  = "gcr.io/k8s-skaffold/skaffold:v0.15.0"
	cloudbuildPath = "cloudbuild.yaml"
)

type CloudbuildTemplate struct {
	ClusterName string
	ClusterZone string
	ProjectName string
	Image       string
}

var tmplt = `steps:
  - name: "{{ .Image }}"
    args: ["gcloud", "container", "clusters", "get-credentials", "{{ .ClusterName }}", "--zone", "{{ .ClusterZone }}", "--project", "{{ .ProjectName }}"]
  - name: "{{ .Image }}"
    args: ["skaffold", "run"]
`

func (c *Client) build() ([]byte, error) {
	t := CloudbuildTemplate{
		ClusterName: c.Cluster.Name,
		ClusterZone: c.Cluster.Zone,
		ProjectName: c.GCPPRoject.Name,
		Image:       skaffoldImage,
	}
	tmpl := template.New("cloudbuild")
	tmpl, err := tmpl.Parse(tmplt)
	if err != nil {
		return nil, errors.Wrap(err, "parsing template")
	}
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, t); err != nil {
		return nil, errors.Wrap(err, "parsing template for cloudbuild")
	}
	return tpl.Bytes(), nil
}

// WriteCloudbuildYaml write the cloudbuild yaml to fp
func (c *Client) WriteCloudbuildYaml() error {
	data, err := c.build()
	if err != nil {
		return err
	}
	if _, err := os.Create(cloudbuildPath); err != nil {
		return err
	}
	return ioutil.WriteFile(cloudbuildPath, data, 0644)
}
