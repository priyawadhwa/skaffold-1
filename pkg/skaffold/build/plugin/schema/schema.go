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

package schema

type PluginManifest struct {
	Name        string     `yaml:"name,omitempty"`
	Platforms   []Platform `yaml:"platforms,omitempty"`
	Version     string     `yaml:"version,omitempty"`
	Description string     `yaml:"description,omitempty"`
	Caveats     string     `yaml:"caveats,omitempty"`
	Flags       []Flag     `yaml:"flags,omitempty"`
}

type Platform struct {
	OS     string `yaml:"os,omitempty"`
	Sha256 string `yaml:"sha256,omitempty"`
	URI    string `yaml:"uri,omitempty"`
}

type Flag struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
}
