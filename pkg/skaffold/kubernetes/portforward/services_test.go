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

package portforward

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
)

func TestRetrieveContainerNameAndPortNameFromPod(t *testing.T) {
	pod := &v1.Pod{
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  "container1",
					Ports: []v1.ContainerPort{retrievePort(1), retrievePort(2)},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container2",
					Ports: []v1.ContainerPort{retrievePort(3), retrievePort(4)},
				},
			},
		},
	}

	tests := []struct {
		description           string
		expectedContainerName string
		expectedPortName      string
		port                  int32
		shouldErr             bool
	}{
		{
			description:           "port matches init container",
			port:                  2,
			expectedContainerName: "container1",
			expectedPortName:      "port2",
		}, {
			description:           "port matches container",
			port:                  3,
			expectedContainerName: "container2",
			expectedPortName:      "port3",
		}, {
			description: "port matches no container",
			port:        500,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actualContainerName, actualPortName, err := retrieveContainerNameAndPortNameFromPod(pod, test.port)
			testutil.CheckError(t, test.shouldErr, err)
			if actualContainerName != test.expectedContainerName {
				t.Fatalf("actual container name doesn't match expected container name. \n Expected: %s \n Acutal: %s", test.expectedContainerName, actualContainerName)
			}
			if actualPortName != test.expectedPortName {
				t.Fatalf("actual port name doesn't match expected port name. \n Expected: %s \n Actual: %s", test.expectedPortName, actualPortName)
			}
		})
	}
}

func retrievePort(i int32) v1.ContainerPort {
	return v1.ContainerPort{
		Name:          fmt.Sprintf("port%d", i),
		ContainerPort: i,
	}
}
