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

package build

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

var defaultArtifactCache = ArtifactCache{
	"key": Artifact{
		ImageName: "image",
		Tag:       "image:tag",
	},
}

func mockHashForArtifact(hashes map[string]string) func(context.Context, *latest.Artifact) (string, error) {
	return func(ctx context.Context, a *latest.Artifact) (string, error) {
		return hashes[a.ImageName], nil
	}
}

func Test_NewCache(t *testing.T) {
	tests := []struct {
		useCache          bool
		updateCacheFile   bool
		name              string
		expectedCache     *Cache
		cacheFileContents interface{}
	}{
		{
			name:              "get a valid cache from file",
			useCache:          true,
			cacheFileContents: defaultArtifactCache,
			updateCacheFile:   true,
			expectedCache: &Cache{
				artifactCache: defaultArtifactCache,
				useCache:      true,
			},
		},
		{
			name:              "valid cache file exists, but useCache is false",
			useCache:          false,
			cacheFileContents: defaultArtifactCache,
			expectedCache:     &Cache{},
		},
		{

			name:              "corrupted cache file",
			useCache:          true,
			cacheFileContents: "corrupted cache file",
			expectedCache:     &Cache{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			cacheFile := createTempCacheFile(t, test.cacheFileContents)

			if test.updateCacheFile {
				test.expectedCache.cacheFile = cacheFile
			}
			actualCache := NewCache(test.useCache, cacheFile)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedCache, actualCache) {
				t.Errorf("Expected result different from actual result. Expected: %v, Actual: %v", test.expectedCache, actualCache)
			}
		})
	}
}

func Test_RetrieveCachedArtifacts(t *testing.T) {
	tests := []struct {
		name                 string
		cache                *Cache
		hashes               map[string]string
		artifacts            []*latest.Artifact
		expectedArtifacts    []*latest.Artifact
		expectedBuildResults []Artifact
	}{
		{
			name:              "useCache is false, return all artifacts",
			cache:             &Cache{},
			artifacts:         []*latest.Artifact{{ImageName: "image1"}},
			expectedArtifacts: []*latest.Artifact{{ImageName: "image1"}},
		},
		{
			name:              "no artifacts in cache",
			cache:             &Cache{useCache: true},
			hashes:            map[string]string{"image1": "hash"},
			artifacts:         []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedArtifacts: []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
		},
		{
			name: "one artifact in cache",
			cache: &Cache{
				useCache: true,
				artifactCache: ArtifactCache{
					"hash": Artifact{
						ImageName: "image1",
						Tag:       "image1:tag",
					},
				},
			},
			hashes:               map[string]string{"image1": "hash"},
			artifacts:            []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedBuildResults: []Artifact{{ImageName: "image1", Tag: "image1:tag"}},
			expectedArtifacts:    []*latest.Artifact{{ImageName: "image2"}},
		},
		{
			name: "both artifacts in cache, but one needs to be rebuilt",
			cache: &Cache{
				useCache: true,
				artifactCache: ArtifactCache{
					"hash": Artifact{
						ImageName: "image1",
						Tag:       "image1:tag",
					},
					"hash2": Artifact{
						ImageName: "image2",
						Tag:       "image2:tag",
					},
				},
			},
			hashes:               map[string]string{"image1": "hash", "image2": "newhash"},
			artifacts:            []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expectedArtifacts:    []*latest.Artifact{{ImageName: "image2"}},
			expectedBuildResults: []Artifact{{ImageName: "image1", Tag: "image1:tag"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalHash := hashForArtifact
			hashForArtifact = mockHashForArtifact(test.hashes)
			defer func() {
				hashForArtifact = originalHash
			}()

			actualArtifacts, actualBuildResults := test.cache.RetrieveCachedArtifacts(context.Background(), os.Stdout, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedArtifacts, actualArtifacts)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedBuildResults, actualBuildResults)
		})
	}
}

func createTempCacheFile(t *testing.T, cacheFileContents interface{}) string {
	temp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("error creating temp cache file: %v", err)
	}
	defer temp.Close()
	contents, err := yaml.Marshal(cacheFileContents)
	if err != nil {
		t.Fatalf("error marshalling cache: %v", err)
	}
	if err := ioutil.WriteFile(temp.Name(), contents, 0755); err != nil {
		t.Fatalf("error writing contents to %s: %v", temp.Name(), err)
	}
	return temp.Name()
}