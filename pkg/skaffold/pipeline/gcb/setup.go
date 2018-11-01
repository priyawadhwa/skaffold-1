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
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/input"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Client holds necessary information for executing the pipeline command
type Client struct {
	APIKey        string
	Token         string
	GithubProject GithubProject
	GCPPRoject    GCPPRoject
	Cluster       Cluster
}

// Cluster holds the necessary information for specifying a GKE cluster
type Cluster struct {
	Name string
	Zone string
}

// GCPPRoject holds info around the GCP project
type GCPPRoject struct {
	Name string `json:"projectId"`
	ID   string `json:"projectNumber"`
}

// NewClient returns a client with the name of a GCP project
// and associated API Key
func NewClient() (*Client, error) {
	project, err := retrieveProject()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving client")
	}
	gcp, err := retrieveGCPPRoject(project)
	if err != nil {
		return nil, errors.Wrap(err, "getting gcp project")
	}
	key, err := retrieveAPIKey(project)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving API Key")
	}
	cluster, err := retrieveCluster()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cluster")
	}
	token, err := retrieveToken()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving token")
	}
	github, err := NewGithubProject()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving github project")
	}
	return &Client{
		GCPPRoject:    *gcp,
		APIKey:        key,
		Token:         token,
		Cluster:       *cluster,
		GithubProject: *github,
	}, nil
}

// SetProject sets the gcloud project
func (c *Client) SetProject() error {
	cmd := exec.Command("gcloud", "config", "set", "project", c.GCPPRoject.Name)
	return util.RunCmd(cmd)
}

func retrieveAPIKey(project string) (string, error) {
	url := "https://console.cloud.google.com/apis/credentials?project=%s"
	url = fmt.Sprintf(url, project)
	msg := `Skaffold needs access to an API Key to interact with the Google Cloud Build API.
Please navigate to %s and enter in an API Key with authorization to interact with the Google Cloud Build API:
`
	msg = fmt.Sprintf(msg, url)
	return input.Password(msg)
}

func retrieveProject() (string, error) {
	projects, err := listProjects()
	if err != nil {
		return "", errors.Wrap(err, "retrieving projects")
	}
	p, err := input.UserSelectedList("GCP project", projects)
	if err != nil {
		return "", errors.Wrap(err, "selecting gcp project")
	}
	cmd := exec.Command("gcloud", "config", "set", "project", p)
	return p, util.RunCmd(cmd)
}

func listProjects() ([]string, error) {
	currentProject := exec.Command("gcloud", "config", "get-value", "project")
	current, err := util.RunCmdOut(currentProject)
	if err != nil {
		return nil, errors.Wrap(err, "getting current project")
	}
	return []string{string(current), constants.Other}, nil
}

func retrieveCluster() (*Cluster, error) {
	clusters, err := listClusters()
	if err != nil {
		return nil, err
	}
	var names []string

	// TODO: Add logic for creating a new cluster
	for _, c := range clusters {
		names = append(names, c.Name)
	}

	var chosen Cluster

	nameQ := []*survey.Question{
		{
			Name: "name",
			Prompt: &survey.Select{
				Message: "Choose a cluster:",
				Options: names,
			},
			Validate:  survey.Required,
			Transform: survey.Title,
		},
	}

	err = survey.Ask(nameQ, &chosen)
	if err != nil {
		return nil, errors.Wrap(err, "getting cluster name")
	}

	chosen.Name = strings.ToLower(chosen.Name)

	var zones []string
	for _, c := range clusters {
		if c.Name == chosen.Name {
			zones = append(zones, c.Zone)
		}
	}

	if len(zones) == 1 {
		chosen.Zone = zones[0]
		return &chosen, nil
	}

	zoneQ := []*survey.Question{
		{
			Name: "zone",
			Prompt: &survey.Select{
				Message: "Choose a zone:",
				Options: zones,
			},
		},
	}

	if err := survey.Ask(zoneQ, &chosen); err != nil {
		return nil, errors.Wrap(err, "getting chosen cluster")
	}

	return &chosen, nil

}

func listClusters() ([]*Cluster, error) {
	list := exec.Command("gcloud", "container", "clusters", "list", "--format=\"json\"")
	data, err := util.RunCmdOut(list)
	if err != nil {
		return nil, errors.Wrap(err, "getting list of clusters")
	}
	var clusters []*Cluster
	if err := json.Unmarshal(data, &clusters); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return clusters, nil
}

func retrieveToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	token, err := util.RunCmdOut(cmd)
	return strings.Trim(string(token), "\n"), err
}

func retrieveGCPPRoject(project string) (*GCPPRoject, error) {
	cmd := exec.Command("gcloud", "projects", "describe", project, "--format=\"json\"")
	data, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting gcp project info")
	}
	var gcp GCPPRoject
	if err := json.Unmarshal(data, &gcp); err != nil {
		return nil, errors.Wrap(err, "unmarshalling gcp project")
	}
	return &gcp, nil
}
