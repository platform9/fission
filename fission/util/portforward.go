/*
Copyright 2016 The Fission Authors.

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

package util

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/fission/fission"
	"github.com/fission/fission/fission/log"
)

// Port forward a free local port to a pod on the cluster. The pod is
// found in the specified namespace by labelSelector. The pod's port
// is found by looking for a service in the same namespace and using
// its targetPort. Once the port forward is started, wait for it to
// start accepting connections before returning.
func SetupPortForward(namespace, labelSelector string) string {
	log.Verbose(2, "Setting up port forward to %s in namespace %s",
		labelSelector, namespace)

	localPort, err := findFreePort()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error finding unused port :%v", err.Error()))
	}

	log.Verbose(2, "Waiting for local port %v", localPort)
	for {
		conn, _ := net.DialTimeout("tcp",
			net.JoinHostPort("", localPort), time.Millisecond)
		if conn != nil {
			conn.Close()
		} else {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	log.Verbose(2, "Starting port forward from local port %v", localPort)
	go func() {
		err := runPortForward(labelSelector, localPort, namespace)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error forwarding to port %v: %s", localPort, err.Error()))
		}
	}()

	log.Verbose(2, "Waiting for port forward %v to start...", localPort)
	for {
		conn, _ := net.DialTimeout("tcp",
			net.JoinHostPort("", localPort), time.Millisecond)
		if conn != nil {
			conn.Close()
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	log.Verbose(2, "Port forward from local port %v started", localPort)

	return localPort
}

func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	err = listener.Close()
	if err != nil {
		return "", err
	}

	return port, nil
}

// runPortForward creates a local port forward to the specified pod
func runPortForward(labelSelector string, localPort string, ns string) error {
	config, clientset := GetKubernetesClient()

	log.Verbose(2, "Connected to Kubernetes API")

	// if namespace is unset, try to find a pod in any namespace
	if len(ns) == 0 {
		ns = meta_v1.NamespaceAll
	}

	// get the pod; if there is more than one, ask the user to disambiguate
	podList, err := clientset.CoreV1().Pods(ns).
		List(meta_v1.ListOptions{LabelSelector: labelSelector})
	if err != nil || len(podList.Items) == 0 {
		log.Fatal(fmt.Sprintf("Error getting pod for port-forwarding with label selector %v: %v", labelSelector, err))
	}

	nsList := make([]string, 0)
	namespaces := make(map[string][]*v1.Pod)

	// make a useful error message if there is more than one install
	if len(podList.Items) > 0 {
		for _, p := range podList.Items {
			if _, ok := namespaces[p.Namespace]; !ok {
				namespaces[p.Namespace] = []*v1.Pod{}
				nsList = append(nsList, p.Namespace)
			}
			namespaces[p.Namespace] = append(namespaces[p.Namespace], &p)
		}
		if len(nsList) > 1 {
			log.Fatal(fmt.Sprintf("Found %v fission installs, set FISSION_NAMESPACE to one of: %v",
				len(namespaces), strings.Join(nsList, " ")))
		}
	}

	// there is at most one namespace in nsList,
	// use index 0 to get from it directly.
	ns = nsList[0]
	pods, ok := namespaces[ns]
	if !ok {
		log.Fatal(fmt.Sprintf("Error finding fission install within the given namespace %v, please check FISSION_NAMESPACE is set properly", ns))
	}

	var podName, podNameSpace string

	// make sure we establish the connection to a healthy pod
	for _, p := range pods {
		if fission.IsReadyPod(p) {
			podName = p.Name
			podNameSpace = p.Namespace
			break
		}
	}

	// get the service and the target port
	svcs, err := clientset.CoreV1().Services(podNameSpace).
		List(meta_v1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		log.Fatal(fmt.Sprintf("Error getting %v service :%v", labelSelector, err.Error()))
	}
	if len(svcs.Items) == 0 {
		log.Fatal(fmt.Sprintf("Service %v not found", labelSelector))
	}
	service := &svcs.Items[0]

	var targetPort string
	for _, servicePort := range service.Spec.Ports {
		targetPort = servicePort.TargetPort.String()
	}
	log.Verbose(2, "Connecting to port %v on pod %v/%v", targetPort, podNameSpace, podNameSpace)

	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	// create request URL
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").
		Namespace(podNameSpace).Name(podName).SubResource("portforward")
	url := req.URL()

	// create ports slice
	portCombo := localPort + ":" + targetPort
	ports := []string{portCombo}

	// actually start the port-forwarding process here
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		msg := fmt.Sprintf("Failed to connect to Fission service on Kubernetes: %v", err.Error())
		log.Fatal(msg)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	outStream := os.Stdout
	if log.Verbosity < 2 {
		outStream = nil
	}
	fw, err := portforward.New(dialer, ports, stopChannel, readyChannel, outStream, os.Stderr)
	if err != nil {
		msg := fmt.Sprintf("portforward.new errored out :%v", err.Error())
		log.Fatal(msg)
	}

	log.Verbose(2, "Starting port forwarder")
	return fw.ForwardPorts()
}
