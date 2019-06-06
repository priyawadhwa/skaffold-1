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
	"sort"
	"strconv"
	"time"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubectl/polymorphichelpers"
	"k8s.io/kubernetes/pkg/kubectl/util/podutils"
)

// RetrieveServices retrieves all services in the cluster matching the given label
func RetrieveServices(label string) ([]v1.Service, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting clientset")
	}
	services, err := clientset.CoreV1().Services("").List(metav1.ListOptions{
		LabelSelector: label,
	})
	return services.Items, err
}

// We will port forward everything from here
// We want to wait on the pod to be created and then port forward
// Use a goroutine
func (p *PortForwarder) portForward(objects []runtime.Object) error {
	return nil
}

func (p *PortForwarder) portForwardServices(ctx context.Context) error {

	services, err := RetrieveServices(p.label)
	if err != nil {
		return errors.Wrap(err, "retrieving services to port forward")
	}

	for _, s := range services {
		for _, port := range s.Spec.Ports {
			fmt.Println("port forwarding service status:", s.Name, s.Status)
			if err := p.portForwardService(ctx, s, port.Port); err != nil {
				logrus.Warnf("unable to port forward service/%s port %v: %v", s.Name, port.Port, err)
			}
		}
	}

	return nil
}

func (p *PortForwarder) portForwardService(ctx context.Context, service v1.Service, port int32) error {
	resourceVersion, err := strconv.Atoi(service.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}

	entry := p.getCurrentEntry("service", service.Name, service.Namespace, port, resourceVersion)
	if entry.port != entry.localPort {
		color.Yellow.Fprintf(p.output, "Forwarding service %s to local port %d.\n", service.Name, entry.localPort)
	}

	sortBy := func(pods []*v1.Pod) sort.Interface { return sort.Reverse(podutils.ActivePods(pods)) }

	config, err := getClientConfig()
	if err != nil {
		return errors.Wrap(err, "getting client config for kubernetes client")
	}

	clientset, err := corev1client.NewForConfig(config)
	if err != nil {
		return err
	}
	_, selector, err := polymorphichelpers.SelectorsForObject(&service)
	if err != nil {
		return errors.Wrap(err, "getting selector for service")
	}

	fmt.Println("got selector", selector.String())
	fmt.Println("passing in", service.Namespace, selector.String())
	pod, _, err := polymorphichelpers.GetFirstPod(clientset, service.Namespace, selector.String(), 5*time.Minute, sortBy)
	if err != nil {
		return errors.Wrap(err, "unable to get forwardable pod")
	}
	entry.podName = pod.Name
	fmt.Println("forwraded pod is ", pod.Name)

	client, err := GetClientset()
	if err != nil {
		return errors.Wrap(err, "getting clienset")
	}
	pods := client.CoreV1().Pods(pod.Namespace)

	if err := WaitForPodRunning(ctx, pods, pod.Name, 5*time.Minute); err != nil {
		return errors.Wrapf(err, "%s never started running", pod.Name)
	}
	return p.forward(ctx, entry)
}
