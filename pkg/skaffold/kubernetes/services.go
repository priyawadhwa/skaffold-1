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

package kubernetes

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RetrieveServicesResources retrieves all services in the cluster matching the given label
// as a list of PortForwardResources
func RetrieveServicesResources(label string) ([]latest.PortForwardResource, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting clientset")
	}
	services, err := clientset.CoreV1().Services("").List(metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "selecting services by label %s", label)
	}
	var resources []latest.PortForwardResource
	for _, s := range services.Items {
		for _, p := range s.Spec.Ports {
			resources = append(resources, latest.PortForwardResource{
				Type:      "service",
				Name:      s.Name,
				Namespace: s.Namespace,
				Port:      p.Port,
			})
		}
	}
	return resources, nil
}

// We will port forward everything from here
// We want to wait on the pod to be created and then port forward
func (p *PortForwarder) portForwardResources(ctx context.Context, resources []latest.PortForwardResource) error {
	for _, r := range resources {
		r := r
		go func() {
			if err := p.portForwardResource(ctx, r); err != nil {
				logrus.Warnf("Unable to port forward %s/%s: %v", r.Type, r.Name, err)
			}
		}()
	}
	return nil
}

func (p *PortForwarder) portForwardResource(ctx context.Context, resource latest.PortForwardResource) error {
	// Get port forward entry for this resource
	entry, err := p.getCurrentEntry(resource)
	if err != nil {
		return errors.Wrapf(err, "getting port forward entry for %s/%s", resource.Type, resource.Name)
	}

	// Forward the resource
	return p.forward(ctx, entry)
}

// retrieveContainerNameAndPortNameFromPod returns the container name and port name for a given port and pod
func retrieveContainerNameAndPortNameFromPod(pod *v1.Pod, port int32) (string, string, error) {
	for _, c := range pod.Spec.InitContainers {
		for _, p := range c.Ports {
			if p.ContainerPort == port {
				return c.Name, p.Name, nil
			}
		}
	}
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			if p.ContainerPort == port {
				return c.Name, p.Name, nil
			}
		}
	}
	return "", "", fmt.Errorf("pod %s does not expose port %d", pod.Name, port)
}
