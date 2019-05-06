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

package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"gopkg.in/yaml.v2"
)

func TestArtifactCache(t *testing.T) {
	// Create temp cache file
	cacheFile, cleanup := testutil.TempFile(t, "", nil)
	defer cleanup()

	// Run skaffold run with the --cache-artifacts flag and --cache-file=cacheFile

	args := []string{"--cache-artifacts=true", fmt.Sprintf("--cache-file=%s", cacheFile)}
	dir := "examples/getting-started"

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Run(args...).WithConfig("").InDir(dir).InNs(ns.Name).WithEnv(nil).RunOrFailOutput(t)
	client.WaitForPodsReady([]string{"getting-started"}...)
	skaffold.Delete().WithConfig("").InDir(dir).InNs(ns.Name).WithEnv(nil).RunOrFail(t)

	// Make sure there is one entry in the cache
	contents, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("error reading cacheFile %s: %v \n", cacheFile, err)
	}

	var artifactCache cache.ArtifactCache
	if err := yaml.Unmarshal(contents, &artifactCache); err != nil {
		t.Fatalf("err unmarshalling artifact cache: %v", err)
	}

	if len(artifactCache) != 1 {
		t.Fatalf("incorrect contents of artifact cache. expected 1 entry, got: \n %v", artifactCache)
	}
}

func TestArtifactCacheRaceCondition(t *testing.T) {
	// Create temp cache file
	cacheFile, cleanup := testutil.TempFile(t, "", nil)
	defer cleanup()

	// Run skaffold run with the --cache-artifacts flag and --cache-file=cacheFile

	args := []string{"--cache-artifacts=true", fmt.Sprintf("--cache-file=%s", cacheFile)}
	dir := "testdata/artifact-cache"
	absPath, err := filepath.Abs("testdata/artifact-cache/foo")
	if err != nil {
		t.Fatalf("error getting absolute path to testdata/artifact-cache/foo: %v", err)
	}
	env := []string{fmt.Sprintf("SKAFFOLD_INTEGRATION_TEST_PREBUILD_COMMAND=echo change > %s", absPath)}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Run(args...).WithConfig("").InDir(dir).InNs(ns.Name).WithEnv(env).RunOrFailOutput(t)
	client.WaitForPodsReady([]string{"artifact-cache"}...)
	skaffold.Delete().WithConfig("").InDir(dir).InNs(ns.Name).WithEnv(env).RunOrFail(t)

	// Make sure there are no entries in the cache
	contents, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("error reading cacheFile %s: %v \n", cacheFile, err)
	}

	var artifactCache cache.ArtifactCache
	if err := yaml.Unmarshal(contents, &artifactCache); err != nil {
		t.Fatalf("err unmarshalling artifact cache: %v", err)
	}

	if len(artifactCache) != 0 {
		t.Fatalf("incorrect contents of artifact cache. expected no entries, got: \n %v", artifactCache)
	}
}
