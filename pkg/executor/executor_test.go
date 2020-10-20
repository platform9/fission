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
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/client"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

// return the number of pods in the given namespace matching the given labels
func countPods(kubeClient *kubernetes.Clientset, ns string, labelz map[string]string) int {
	pods, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelz).AsSelector().String(),
	})
	if err != nil {
		log.Panicf("Failed to list pods: %v", err)
	}
	return len(pods.Items)
}

func createTestNamespace(kubeClient *kubernetes.Clientset, ns string) {
	_, err := kubeClient.CoreV1().Namespaces().Create(&apiv1.Namespace{
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
	svc, err := kubeClient.CoreV1().Services(ns).Create(&apiv1.Service{
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

func TestExecutor(t *testing.T) {
	// run in a random namespace so we can have concurrent tests
	// on a given cluster
	rand.Seed(time.Now().UTC().UnixNano())
	testID := rand.Intn(999)
	fissionNs := fmt.Sprintf("test-%v", testID)
	functionNs := fmt.Sprintf("test-function-%v", testID)

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
	defer kubeClient.CoreV1().Namespaces().Delete(fissionNs, nil)

	createTestNamespace(kubeClient, functionNs)
	defer kubeClient.CoreV1().Namespaces().Delete(functionNs, nil)

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := config.Build()
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
	env, err := fissionClient.CoreV1().Environments(fissionNs).Create(&fv1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nodejs",
			Namespace: fissionNs,
		},
		Spec: fv1.EnvironmentSpec{
			Version: 1,
			Runtime: fv1.Runtime{
				Image: "fission/node-env",
			},
			Builder: fv1.Builder{},
		},
	})
	if err != nil {
		log.Panicf("failed to create env: %v", err)
	}

	// create poolmgr
	port := 9999
	err = StartExecutor(logger, functionNs, "fission-builder", port)
	if err != nil {
		log.Panicf("failed to start poolmgr: %v", err)
	}

	// connect poolmgr client
	poolmgrClient := client.MakeClient(logger, fmt.Sprintf("http://localhost:%v", port))

	// Wait for pool to be created (we don't actually need to do
	// this, since the API should do the right thing in any case).
	// waitForPool(functionNs, "nodejs")
	time.Sleep(6 * time.Second)

	envRef := fv1.EnvironmentReference{
		Namespace: env.ObjectMeta.Namespace,
		Name:      env.ObjectMeta.Name,
	}

	deployment := fv1.Archive{
		Type:    fv1.ArchiveTypeLiteral,
		Literal: []byte(`module.exports = async function(context) { return { status: 200, body: "Hello, world!\n" }; }`),
	}

	// create a package
	p := &fv1.Package{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: fissionNs,
		},
		Spec: fv1.PackageSpec{
			Environment: envRef,
			Deployment:  deployment,
		},
	}
	p, err = fissionClient.CoreV1().Packages(fissionNs).Create(p)
	if err != nil {
		log.Panicf("failed to create package: %v", err)
	}

	// create a function
	f := &fv1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: fissionNs,
		},
		Spec: fv1.FunctionSpec{
			Environment: envRef,
			Package: fv1.FunctionPackageRef{
				PackageRef: fv1.PackageRef{
					Namespace:       p.ObjectMeta.Namespace,
					Name:            p.ObjectMeta.Name,
					ResourceVersion: p.ObjectMeta.ResourceVersion,
				},
			},
		},
	}
	_, err = fissionClient.CoreV1().Functions(fissionNs).Create(f)
	if err != nil {
		log.Panicf("failed to create function: %v", err)
	}

	// create a service to call fetcher and the env container
	labels := map[string]string{"functionName": f.ObjectMeta.Name}
	var fetcherPort int32 = 30001
	fetcherSvc := createSvc(kubeClient, functionNs, fmt.Sprintf("%v-%v", f.ObjectMeta.Name, "fetcher"), 8000, fetcherPort, labels)
	defer kubeClient.CoreV1().Services(functionNs).Delete(fetcherSvc.ObjectMeta.Name, nil)

	var funcSvcPort int32 = 30002
	functionSvc := createSvc(kubeClient, functionNs, f.ObjectMeta.Name, 8888, funcSvcPort, labels)
	defer kubeClient.CoreV1().Services(functionNs).Delete(functionSvc.ObjectMeta.Name, nil)

	// the main test: get a service for a given function
	t1 := time.Now()
	svc, err := poolmgrClient.GetServiceForFunction(context.Background(), &f.ObjectMeta)
	if err != nil {
		log.Panicf("failed to get func svc: %v", err)
	}
	log.Printf("svc for function created at: %v (in %v)", svc, time.Since(t1))

	// ensure that a pod with the label functionName=f.ObjectMeta.Name exists
	podCount := countPods(kubeClient, functionNs, map[string]string{"functionName": f.ObjectMeta.Name})
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
