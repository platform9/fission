package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return "", nil
	}

	err = listener.Close()
	if err != nil {
		return "", err
	}

	err = file.Close()
	if err != nil {
		return "", err
	}

	return port, nil
}

// runPortForward creates a local port forward to the specified pod
func runPortForward(kubeConfig string, labelSelector string, localPort string, fissionNamespace string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		fatal(fmt.Sprintf("Failed to connect to Kubernetes: %s", err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fatal(fmt.Sprintf("Failed to connect to Kubernetes: %s", err))
	}

	verbose(2, "Connected to Kubernetes API")

	// if fission namespace is unset, try to find a fission pod in any namespace
	if len(fissionNamespace) == 0 {
		fissionNamespace = meta_v1.NamespaceAll
	}

	// get the pod; if there is more than one, ask the user to disambiguate
	podList, err := clientset.CoreV1().Pods(fissionNamespace).
		List(meta_v1.ListOptions{LabelSelector: labelSelector})
	if err != nil || len(podList.Items) == 0 {
		fatal("Error getting controller pod for port-forwarding")
	}

	// make a useful error message if there is more than one install
	if len(podList.Items) > 1 {
		namespaces := make([]string, 0)
		for _, p := range podList.Items {
			namespaces = append(namespaces, p.Namespace)
		}
		fatal(fmt.Sprintf("Found %v fission installs, set FISSION_NAMESPACE to one of: %v",
			len(podList.Items), strings.Join(namespaces, " ")))
	}

	// pick the first pod
	podName := podList.Items[0].Name
	podNameSpace := podList.Items[0].Namespace

	// get the service and the target port
	svcs, err := clientset.CoreV1().Services(podNameSpace).
		List(meta_v1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		fatal(fmt.Sprintf("Error getting %v service :%v", labelSelector, err.Error()))
	}
	if len(svcs.Items) == 0 {
		fatal(fmt.Sprintf("Service %v not found", labelSelector))
	}
	service := &svcs.Items[0]

	var targetPort string
	for _, servicePort := range service.Spec.Ports {
		targetPort = servicePort.TargetPort.String()
	}
	verbose(2, "Connecting to port %v on pod %v/%v", targetPort, podNameSpace, podNameSpace)

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
		msg := fmt.Sprintf("spdy round tripper errored out :%v", err.Error())
		fatal(msg)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	outStream := os.Stdout
	if verbosity < 2 {
		outStream = nil
	}
	fw, err := portforward.New(dialer, ports, stopChannel, readyChannel, outStream, os.Stderr)
	if err != nil {
		msg := fmt.Sprintf("portforward.new errored out :%v", err.Error())
		fatal(msg)
	}

	verbose(2, "Starting port forwarder")
	return fw.ForwardPorts()
}

// Port forward a free local port to a pod on the cluster. The pod is
// found in the specified namespace by labelSelector. The pod's port
// is found by looking for a service in the same namespace and using
// its targetPort. Once the port forward is started, wait for it to
// start accepting connections before returning.
func setupPortForward(kubeConfig, namespace, labelSelector string) string {
	verbose(2, "Setting up port forward to %s in namespace %s using the kubeconfig at %s",
		labelSelector, namespace, kubeConfig)

	localPort, err := findFreePort()
	if err != nil {
		fatal(fmt.Sprintf("Error finding unused port :%v", err.Error()))
	}

	verbose(2, "Waiting for local port %v", localPort)
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

	verbose(2, "Starting port forward from local port %v", localPort)
	go func() {
		err := runPortForward(kubeConfig, labelSelector, localPort, namespace)
		if err != nil {
			fatal(fmt.Sprintf("Error forwarding to controller port: %s", err.Error()))
		}
	}()

	verbose(2, "Waiting for port forward %v to start...", localPort)
	for {
		conn, _ := net.DialTimeout("tcp",
			net.JoinHostPort("", localPort), time.Millisecond)
		if conn != nil {
			conn.Close()
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	verbose(2, "Port forward from local port %v started", localPort)

	return localPort
}
