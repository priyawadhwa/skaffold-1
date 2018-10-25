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
	"encoding/json"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/constants"
	"github.com/pkg/errors"
)

type BuildTrigger struct {
	Description string `json:"description"`
	Github      Github `json:"github"`
	Filename    string `json:"filename"`
}

type Github struct {
	URL  string `json:"url"`
	Push Push   `json:"push"`
}

type Push struct {
	Branch string `json:"branch"`
}

// CreateBuildTrigger creates a build trigger in the specified gcp project
func (c *Client) CreateBuildTrigger() error {
	tail := fmt.Sprintf("v1/projects/%s/triggers", c.GCPPRoject.Name)
	url := fmt.Sprintf("https://github.com/%s/%s", c.GithubProject.Organization, c.GithubProject.Repo)
	trigger := BuildTrigger{
		Description: "build trigger for skaffold run",
		Github: Github{
			URL: url,
			Push: Push{
				Branch: "master",
			},
		},
		Filename: "cloudbuild.yaml",
	}
	data, err := json.Marshal(trigger)
	if err != nil {
		return errors.Wrap(err, "marshalling trigger")
	}
	buf := bytes.NewBuffer(data)
	_, err = c.Request(constants.POST, tail, buf)
	return err
}
