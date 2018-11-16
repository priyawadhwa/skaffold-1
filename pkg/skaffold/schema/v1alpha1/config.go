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

package v1alpha1

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"

	yaml "gopkg.in/yaml.v2"
)

const Version string = "skaffold/v1alpha1"

// NewSkaffoldPipeline creates a SkaffoldPipeline
func NewSkaffoldPipeline() util.VersionedConfig {
	return new(SkaffoldPipeline)
}

// SkaffoldPipeline is the top level config object
// that is parsed from a skaffold.yaml
type SkaffoldPipeline struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build  BuildConfig  `yaml:"build"`
	Deploy DeployConfig `yaml:"deploy"`
}

func (config *SkaffoldPipeline) GetVersion() string {
	return config.APIVersion
}

// GetBuilderName returns the name of the chosen builder
func (config *SkaffoldPipeline) GetBuilderName() string {
	if config.Build.GoogleCloudBuild != nil {
		return constants.GoogleCloudBuilderName
	}
	// default
	return constants.LocalBuilderName
}

// GetDeployerName returns the name of the chosen deployer
func (config *SkaffoldPipeline) GetDeployerName() string {
	if config.Deploy.HelmDeploy != nil {
		return constants.HelmDeployerName
	}
	// default
	return constants.KubectlDeployerName
}

// GetTaggerName returns the name of the chosen tagger
func (config *SkaffoldPipeline) GetTaggerName() string {
	return constants.ShaTaggerName
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts"`
	TagPolicy string      `yaml:"tagPolicy,omitempty"`
	BuildType `yaml:",inline"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local,omitempty"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	SkipPush *bool `yaml:"skipPush,omitempty"`
}

type GoogleCloudBuild struct {
	ProjectID string `yaml:"projectId"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	Name       string `yaml:"name,omitempty"`
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy    *HelmDeploy    `yaml:"helm,omitempty"`
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests []Manifest `yaml:"manifests"`
}

type Manifest struct {
	Paths      []string          `yaml:"paths"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
}

type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases"`
}

type HelmRelease struct {
	Name           string            `yaml:"name"`
	ChartPath      string            `yaml:"chartPath"`
	ValuesFilePath string            `yaml:"valuesFilePath"`
	Values         map[string]string `yaml:"values"`
	Namespace      string            `yaml:"namespace"`
	Version        string            `yaml:"version"`
}

// Artifact represents items that need should be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName      string             `yaml:"imageName"`
	DockerfilePath string             `yaml:"dockerfilePath,omitempty"`
	Workspace      string             `yaml:"workspace"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
}

// DefaultDevSkaffoldPipeline is a partial set of defaults for the SkaffoldPipeline
// when dev mode is specified.
// Each API is responsible for setting its own defaults that are not top level.
var defaultDevSkaffoldPipeline = &SkaffoldPipeline{
	Build: BuildConfig{
		TagPolicy: constants.DefaultDevTagStrategy,
	},
}

// DefaultRunSkaffoldPipeline is a partial set of defaults for the SkaffoldPipeline
// when run mode is specified.
// Each API is responsible for setting its own defaults that are not top level.
var defaultRunSkaffoldPipeline = &SkaffoldPipeline{
	Build: BuildConfig{
		TagPolicy: constants.DefaultRunTagStrategy,
	},
}

// Parse reads from an io.Reader and unmarshals the result into a SkaffoldPipeline.
// The default config argument provides default values for the config,
// which can be overridden if present in the config file.
func (config *SkaffoldPipeline) Parse(contents []byte, useDefault bool) error {
	if useDefault {
		*config = *config.getDefaultForMode(false)
	} else {
		*config = SkaffoldPipeline{}
	}

	return yaml.UnmarshalStrict(contents, config)
}

func (config *SkaffoldPipeline) getDefaultForMode(dev bool) *SkaffoldPipeline {
	if dev {
		return defaultDevSkaffoldPipeline
	}
	return defaultRunSkaffoldPipeline
}
