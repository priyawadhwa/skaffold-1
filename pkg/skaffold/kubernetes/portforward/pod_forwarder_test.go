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
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAutomaticPortForwardPod(t *testing.T) {
	var tests = []struct {
		description     string
		pods            []*v1.Pod
		forwarder       *testForwarder
		expectedPorts   map[int32]bool
		expectedEntries map[string]*portForwardEntry
		availablePorts  []int
		shouldErr       bool
	}{
		{
			description: "single container port",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					portName:  "portname",
					localPort: 8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "unavailable container port",
			expectedPorts: map[int32]bool{
				9000: true,
			},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					containerName: "containername",
					portName:      "portname",
					localPort:     9000,
				},
			},
			availablePorts: []int{9000},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description:     "bad resource version",
			expectedPorts:   map[int32]bool{},
			shouldErr:       true,
			expectedEntries: map[string]*portForwardEntry{},
			availablePorts:  []int{8080},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "10000000000a",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "forward error",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			forwarder:      newTestForwarder(fmt.Errorf(""), true),
			shouldErr:      true,
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					portName:        "portname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					localPort: 8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "two different container ports",
			expectedPorts: map[int32]bool{
				8080:  true,
				50051: true,
			},
			availablePorts: []int{8080, 50051},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					portName:  "portname",
					localPort: 8080,
				},
				"containername2-namespace2-portname2-50051": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname2",
						Namespace: "namespace2",
						Port:      50051,
					},
					portName:  "portname2",
					localPort: 50051,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname2",
						ResourceVersion: "1",
						Namespace:       "namespace2",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 50051,
										Name:          "portname2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "two same container ports",
			expectedPorts: map[int32]bool{
				8080: true,
				9000: true,
			},
			availablePorts: []int{8080, 9000},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 1,
					podName:         "podname",
					containerName:   "containername",
					portName:        "portname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					localPort: 8080,
				},
				"containername2-namespace2-portname2-8080": {
					resourceVersion: 1,
					podName:         "podname2",
					containerName:   "containername2",
					portName:        "portname2",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname2",
						Namespace: "namespace2",
						Port:      8080,
					},
					localPort: 9000,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname2",
						ResourceVersion: "1",
						Namespace:       "namespace2",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername2",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "updated pod gets port forwarded",
			expectedPorts: map[int32]bool{
				8080: true,
			},
			availablePorts: []int{8080},
			expectedEntries: map[string]*portForwardEntry{
				"containername-namespace-portname-8080": {
					resourceVersion: 2,
					podName:         "podname",
					containerName:   "containername",
					portName:        "portname",
					resource: latest.PortForwardResource{
						Type:      "pod",
						Name:      "podname",
						Namespace: "namespace",
						Port:      8080,
					},
					localPort: 8080,
				},
			},
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "1",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "podname",
						ResourceVersion: "2",
						Namespace:       "namespace",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "containername",
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 8080,
										Name:          "portname",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			taken := map[int]struct{}{}

			forwardingPollTime = time.Second
			t.Override(&retrieveAvailablePort, mockRetrieveAvailablePort(taken, test.availablePorts))

			p := NewAutomaticPodForwarder(ioutil.Discard, kubernetes.NewImageList(), []string{""})
			if test.forwarder == nil {
				test.forwarder = newTestForwarder(nil, true)
			}
			p.Forwarder = test.forwarder

			for _, pod := range test.pods {
				err := p.portForwardPod(context.Background(), pod)
				t.CheckError(test.shouldErr, err)
			}

			// Error is already checked above
			t.CheckDeepEqual(test.expectedPorts, test.forwarder.forwardedPorts)

			// cmp.Diff cannot access unexported fields, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expectedEntries, test.forwarder.forwardedEntries) {
				t.Errorf("Forwarded entries differs from expected entries. Expected: %s, Actual: %s", test.expectedEntries, test.forwarder.forwardedEntries)
			}
		})
	}
}

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
