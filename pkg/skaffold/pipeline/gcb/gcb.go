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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	url = "https://cloudbuild.googleapis.com/"
)

var requiredCloudAPIs = []string{
	"container.googleapis.com",
	"cloudbuild.googleapis.com",
}

// EnableRequiredCloudAPIs enables requiredCloudAPIs
func EnableRequiredCloudAPIs() error {
	for _, r := range requiredCloudAPIs {
		logrus.Debugf("Enabling required cloud API %s", r)
		cmd := exec.Command("gcloud", "services", "enable", r)
		if err := util.RunCmd(cmd); err != nil {
			return errors.Wrapf(err, "enabling cloud API %s", r)
		}
	}
	return nil
}

// SetServiceAccountPermissions gives the cloudbuild service account access to GKE clusters
func (c *Client) SetServiceAccountPermissions() error {
	account := fmt.Sprintf("%s@cloudbuild.gserviceaccount.com", c.GCPPRoject.ID)
	cmd := exec.Command("gcloud", "projects", "add-iam-policy-binding", c.GCPPRoject.Name,
		"--member", fmt.Sprintf("serviceAccount:%s", account),
		"--role=roles/container.admin")
	return cmd.Run()
}

// BuildTriggerAuth checks to make sure auth to create the build trigger is present
// if not, it redirects to the GCB GitHub App and polls ListGithubRepositories until the repo shows up
func (c *Client) BuildTriggerAuth() error {
	if c.repoExists() {
		return nil
	}
	return c.githubAppFlow()
}

func (c *Client) githubAppFlow() error {
	url := "https://github.com/apps/google-cloud-build"
	color.Default.Fprintf(os.Stdout, "To use Google Cloud Build with GitHub, please install the GCB GitHub app on your github repository here: %s\n", url)
	return wait.PollImmediate(time.Second*2, 20*time.Minute, func() (bool, error) {
		if c.repoExists() {
			color.Green.Fprintln(os.Stdout, "GitHub App Installation successful! Moving forward with setup...")
			return true, nil
		}
		return false, nil
	})
}

func (c *Client) repoExists() bool {
	resp, err := c.ListGithubRepositories()
	if err != nil {
		return false
	}
	fullName := fmt.Sprintf("%s/%s", c.GithubProject.Organization, c.GithubProject.Repo)
	for _, r := range resp.Repos {
		if r.FullName == fullName {
			return true
		}
	}
	return false
}

type ListGithubRepositoriesResponse struct {
	Repos []Repo `json:"repos"`
}

// Repo represents github repos available to cloud build
type Repo struct {
	InstallationID string `json:"installationId"`
	Name           string `json:"name"`
	FullName       string `json:"fullName"`
}

// ListGithubRepositories returns list of github repos the gcp project has access to
func (c *Client) ListGithubRepositories() (*ListGithubRepositoriesResponse, error) {
	tail := fmt.Sprintf("v1/projects/%s/github/repos", c.GCPPRoject.ID)
	r, err := c.Request(constants.POST, tail, nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting github repos")
	}
	var resp ListGithubRepositoriesResponse
	err = json.Unmarshal(r, &resp)
	return &resp, err
}

// Request creates a request for interacting with the CloudBuild API
func (c *Client) Request(method, tail string, body io.Reader) ([]byte, error) {
	url := url + tail
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "generating http request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)
	req.Header.Set("x-google-project-override", "apikey")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("invalid status code %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}
