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

package acr

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Builder struct {
	*latest.AzureContainerBuild
}

// Creates a new builder with the Azure Container config
func NewBuilder(cfg *latest.AzureContainerBuild) *Builder {
	return &Builder{
		AzureContainerBuild: cfg,
	}
}

// Labels specific to Azure Container Build
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "azure-container-build",
	}
}
