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

package generatepipeline

import (
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateProfile(t *testing.T) {
	var tests = []struct {
		description     string
		skaffoldConfig  *latest.SkaffoldConfig
		expectedProfile *latest.Profile
		responses       []string
		shouldErr       bool
	}{
		{
			description: "successful profile generation docker",
			skaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test",
								ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{},
								},
							},
						},
					},
				},
			},
			expectedProfile: &latest.Profile{
				Name: "oncluster",
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latest.ArtifactType{
									KanikoArtifact: &latest.KanikoArtifact{
										BuildContext: &latest.KanikoBuildContext{
											GCSBucket: "skaffold-kaniko",
										},
									},
								},
							},
						},
						BuildType: latest.BuildType{
							Cluster: &latest.ClusterDetails{
								PullSecretName: "kaniko-secret",
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "successful profile generation jib",
			skaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test",
								ArtifactType: latest.ArtifactType{
									JibMavenArtifact: &latest.JibMavenArtifact{
										Module:  "test-module",
										Profile: "test-profile",
									},
									DockerArtifact: nil,
								},
							},
						},
					},
				},
			},
			expectedProfile: &latest.Profile{
				Name: "oncluster",
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latest.ArtifactType{
									JibMavenArtifact: &latest.JibMavenArtifact{
										Module:  "test-module",
										Profile: "test-profile",
									},
								},
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "failed profile generation",
			skaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{},
					},
				},
			},
			expectedProfile: nil,
			shouldErr:       true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			profile, err := generateProfile(ioutil.Discard, test.skaffoldConfig)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedProfile, profile)
		})
	}
}
