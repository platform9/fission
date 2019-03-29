//
// This test depends on several env vars:
//
//   KUBECONFIG has to point at a kube config with a cluster. The test
//      will use the default context from that config. Be careful,
//      don't point this at your production environment. The test is
//      skipped if KUBECONFIG is undefined.
//
//   TEST_SPECIALIZE_URL
//   TEST_FETCHER_URL
//      These need to point at <node ip>:30001 and <node ip>:30002,
//      where <node ip> is the address of any node in the test
//      cluster.
//
//   FETCHER_IMAGE
//      Optional. Set this to a fetcher image; otherwise uses the
//      default.
//

// Here's how I run this on my setup, with minikube:
// TEST_SPECIALIZE_URL=http://192.168.99.100:30002/specialize TEST_FETCHER_URL=http://192.168.99.100:30001 FETCHER_IMAGE=minikube/fetcher:testing KUBECONFIG=/Users/soam/.kube/config go test -v .

package executor

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/client"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

// return the number of pods in the given namespace matching the given labels
func countPods(kubeClient *kubernetes.Clientset, ns string, labelz map[string]string) int {
	pods, err := kubeClient.Pods(ns).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelz).AsSelector().String(),
	})
	if err != nil {
		log.Panicf("Failed to list pods: %v", err)
	}
	return len(pods.Items)
}

func createTestNamespace(kubeClient *kubernetes.Clientset, ns string) {
	_, err := kubeClient.Namespaces().Create(&apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if err != nil {
		log.Panicf("failed to create ns %v: %v", ns, err)
	}
	log.Printf("Created namespace %v", ns)
}

// create a nodeport service
func createSvc(kubeClient *kubernetes.Clientset, ns string, name string, targetPort int, nodePort int32, labels map[string]string) *apiv1.Service {
	svc, err := kubeClient.Services(ns).Create(&apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeNodePort,
			Ports: []apiv1.ServicePort{
				{
					Protocol:   apiv1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(targetPort),
					NodePort:   nodePort,
				},
			},
			Selector: labels,
		},
	})
	if err != nil {
		log.Panicf("Failed to create svc: %v", err)
	}
	return svc
}

func httpGet(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Panicf("HTTP Get failed: URL %v: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("HTTP Get failed to read body: URL %v: %v", url, err)
	}
	return string(body)
}

func TestExecutor(t *testing.T) {
	// run in a random namespace so we can have concurrent tests
	// on a given cluster
	rand.Seed(time.Now().UTC().UnixNano())
	testId := rand.Intn(999)
	fissionNs := fmt.Sprintf("test-%v", testId)
	functionNs := fmt.Sprintf("test-function-%v", testId)

	// skip test if no cluster available for testing
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) == 0 {
		t.Skip("Skipping test, no kubernetes cluster")
		return
	}

	// connect to k8s
	// and get CRD client
	fissionClient, kubeClient, apiExtClient, err := crd.MakeFissionClient()
	if err != nil {
		log.Panicf("failed to connect: %v", err)
	}

	// create the test's namespaces
	createTestNamespace(kubeClient, fissionNs)
	defer kubeClient.Namespaces().Delete(fissionNs, nil)

	createTestNamespace(kubeClient, functionNs)
	defer kubeClient.Namespaces().Delete(functionNs, nil)

	logger, err := zap.NewDevelopment()
	panicIf(err)

	// make sure CRD types exist on cluster
	err = crd.EnsureFissionCRDs(logger, apiExtClient)
	if err != nil {
		log.Panicf("failed to ensure crds: %v", err)
	}

	err = fissionClient.WaitForCRDs()
	if err != nil {
		log.Panicf("failed to wait crds: %v", err)
	}

	// create an env on the cluster
	env, err := fissionClient.Environments(fissionNs).Create(&crd.Environment{
		Metadata: metav1.ObjectMeta{
			Name:      "nodejs",
			Namespace: fissionNs,
		},
		Spec: fission.EnvironmentSpec{
			Version: 1,
			Runtime: fission.Runtime{
				Image: "fission/node-env",
			},
			Builder: fission.Builder{},
		},
	})
	if err != nil {
		log.Panicf("failed to create env: %v", err)
	}

	// create poolmgr
	port := 9999
	err = StartExecutor(logger, fissionNs, functionNs, "fission-builder", port)
	if err != nil {
		log.Panicf("failed to start poolmgr: %v", err)
	}

	// connect poolmgr client
	poolmgrClient := client.MakeClient(logger, fmt.Sprintf("http://localhost:%v", port))

	// Wait for pool to be created (we don't actually need to do
	// this, since the API should do the right thing in any case).
	// waitForPool(functionNs, "nodejs")
	time.Sleep(6 * time.Second)

	envRef := fission.EnvironmentReference{
		Namespace: env.Metadata.Namespace,
		Name:      env.Metadata.Name,
	}

	deployment := fission.Archive{
		Type:    fission.ArchiveTypeLiteral,
		Literal: []byte(`module.exports = async function(context) { return { status: 200, body: "Hello, world!\n" }; }`),
	}

	// create a package
	p := &crd.Package{
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: fissionNs,
		},
		Spec: fission.PackageSpec{
			Environment: envRef,
			Deployment:  deployment,
		},
	}
	p, err = fissionClient.Packages(fissionNs).Create(p)
	if err != nil {
		log.Panicf("failed to create package: %v", err)
	}

	// create a function
	f := &crd.Function{
		Metadata: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: fissionNs,
		},
		Spec: fission.FunctionSpec{
			Environment: envRef,
			Package: fission.FunctionPackageRef{
				PackageRef: fission.PackageRef{
					Namespace:       p.Metadata.Namespace,
					Name:            p.Metadata.Name,
					ResourceVersion: p.Metadata.ResourceVersion,
				},
			},
		},
	}
	_, err = fissionClient.Functions(fissionNs).Create(f)
	if err != nil {
		log.Panicf("failed to create function: %v", err)
	}

	// create a service to call fetcher and the env container
	labels := map[string]string{"functionName": f.Metadata.Name}
	var fetcherPort int32 = 30001
	fetcherSvc := createSvc(kubeClient, functionNs, fmt.Sprintf("%v-%v", f.Metadata.Name, "fetcher"), 8000, fetcherPort, labels)
	defer kubeClient.Services(functionNs).Delete(fetcherSvc.ObjectMeta.Name, nil)

	var funcSvcPort int32 = 30002
	functionSvc := createSvc(kubeClient, functionNs, f.Metadata.Name, 8888, funcSvcPort, labels)
	defer kubeClient.Services(functionNs).Delete(functionSvc.ObjectMeta.Name, nil)

	// the main test: get a service for a given function
	t1 := time.Now()
	svc, err := poolmgrClient.GetServiceForFunction(context.Background(), &f.Metadata)
	if err != nil {
		log.Panicf("failed to get func svc: %v", err)
	}
	log.Printf("svc for function created at: %v (in %v)", svc, time.Since(t1))

	// ensure that a pod with the label functionName=f.Metadata.Name exists
	podCount := countPods(kubeClient, functionNs, map[string]string{"functionName": f.Metadata.Name})
	if podCount != 1 {
		log.Panicf("expected 1 function pod, found %v", podCount)
	}

	// call the service to ensure it works

	// wait for a bit

	// tap service to simulate calling it again

	// make sure the same pod is still there

	// wait for idleTimeout to ensure the pod is removed

	// remove env

	// wait for pool to be destroyed

	// that's it
}
