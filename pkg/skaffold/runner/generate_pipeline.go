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

package runner

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"

	"github.com/pkg/errors"

	pipeline "github.com/GoogleContainerTools/skaffold/pkg/skaffold/generate_pipeline"
)

func (r *SkaffoldRunner) GeneratePipeline(ctx context.Context, out io.Writer, config *latest.SkaffoldConfig, configPaths []string, fileOut string) error {
	// Keep track of files, configs, and profiles. This will be used to know which files to write
	// profiles to and what flags to add to task commands
	baseConfig := []*pipeline.ConfigFile{
		{
			Path:    r.runCtx.Opts.ConfigurationFile,
			Config:  config,
			Profile: nil,
		},
	}
	configFiles, err := setupConfigFiles(configPaths)
	if err != nil {
		return errors.Wrap(err, "setting up ConfigFiles")
	}
	configFiles = append(baseConfig, configFiles...)

	// Will run the profile setup multiple times and require user input for each specified config
	color.Default.Fprintln(out, "Running profile setup...")
	for _, configFile := range configFiles {
		if err := pipeline.CreateSkaffoldProfile(out, r.runCtx, configFile); err != nil {
			return errors.Wrap(err, "setting up profile")
		}
	}

	color.Default.Fprintln(out, "Generating Pipeline...")
	pipelineYaml, err := pipeline.Yaml(out, r.runCtx, configFiles)
	if err != nil {
		return errors.Wrap(err, "generating pipeline yaml contents")
	}

	// write all yaml pieces to output
	return ioutil.WriteFile(fileOut, pipelineYaml.Bytes(), 0755)
}

func setupConfigFiles(configPaths []string) ([]*pipeline.ConfigFile, error) {
	if configPaths == nil {
		return []*pipeline.ConfigFile{}, nil
	}

	// Read all given config files to read contents into SkaffoldConfig
	var configFiles []*pipeline.ConfigFile
	for _, path := range configPaths {
		parsed, err := schema.ParseConfig(path, true)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing config %s", path)
		}
		config := parsed.(*latest.SkaffoldConfig)

		if err := defaults.Set(config); err != nil {
			return nil, errors.Wrap(err, "setting default values for extra configs")
		}

		configFile := &pipeline.ConfigFile{
			Path:   path,
			Config: config,
		}
		configFiles = append(configFiles, configFile)
	}

	return configFiles, nil
}
