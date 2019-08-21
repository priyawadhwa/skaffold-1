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

package sync

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// PerformDockerComposeSync performs a docker-compose sync
func PerformDockerComposeSync(ctx context.Context, image string, files syncMap, cmdFn func(context.Context, string, syncMap) ([]byte, error)) error {
	if len(files) == 0 {
		return nil
	}
	if output, err := cmdFn(ctx, image, files); err != nil {
		return errors.Wrapf(err, "syncing: %s", string(output))
	}

	return nil
}

func (s *containerSyncer) deleteFileFn(ctx context.Context, image string, files syncMap) ([]byte, error) {
	containerName := "python_app_cloud-run-playground-242117_1"
	if strings.Contains(image, "datastore") {
		containerName = "python_app_datastore_1"
	}
	args := make([]string, 0, 6+len(files))
	args = append(args, "exec", containerName, "--", "rm", "-rf", "--")
	for _, dsts := range files {
		args = append(args, dsts...)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	fmt.Println(cmd)
	return cmd.CombinedOutput()
}

func (s *containerSyncer) copyFileFn(ctx context.Context, image string, files syncMap) ([]byte, error) {
	containerName := "python_app_cloud-run-playground-242117_1"
	if strings.Contains(image, "datastore") {
		containerName = "python_app_datastore_1"
	}
	for file, dst := range files {
		for _, d := range dst {
			cmd := exec.CommandContext(ctx, "docker", "cp", file, fmt.Sprintf("%s:%s", containerName, d))
			fmt.Println(cmd)
			if output, err := cmd.CombinedOutput(); err != nil {
				return output, err
			}
		}
	}
	return nil, nil
}
