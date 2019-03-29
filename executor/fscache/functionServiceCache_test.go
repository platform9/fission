package fscache

import (
	"log"
	"testing"
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

func TestFunctionServiceCache(t *testing.T) {
	logger, err := zap.NewDevelopment()
	panicIf(err)

	fsc := MakeFunctionServiceCache(logger)
	if fsc == nil {
		log.Panicf("error creating cache")
	}

	var fsvc *FuncSvc
	now := time.Now()

	objects := []apiv1.ObjectReference{
		{
			Kind:       "pod",
			Name:       "xxx",
			APIVersion: "v1",
			Namespace:  "fission-function",
		},
		{
			Kind:       "pod",
			Name:       "xxx2",
			APIVersion: "v1",
			Namespace:  "fission-function",
		},
	}

	fsvc = &FuncSvc{
		Function: &metav1.ObjectMeta{
			Name: "foo",
			UID:  "1212",
		},
		Environment: &crd.Environment{
			Metadata: metav1.ObjectMeta{
				Name: "foo-env",
				UID:  "2323",
			},
			Spec: fission.EnvironmentSpec{
				Version: 1,
				Runtime: fission.Runtime{
					Image: "fission/foo-env",
				},
				Builder: fission.Builder{},
			},
		},
		Address:           "xxx",
		KubernetesObjects: objects,
		Ctime:             now,
		Atime:             now,
	}
	_, err := fsc.Add(*fsvc)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to add fsvc: %v", err)
	}

	f, err := fsc.GetByFunction(fsvc.Function)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to get fsvc: %v", err)
	}
	f, err = fsc.GetByFunctionUID(fsvc.Function.UID)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to get fsvc by function uid: %v", err)
	}
	fsvc.Atime = f.Atime
	fsvc.Ctime = f.Ctime
	if f.Address != fsvc.Address {
		fsc.Log()
		log.Panicf("Incorrect fsvc \n(expected: %#v)\n (found: %#v)", fsvc, f)
	}

	err = fsc.TouchByAddress(fsvc.Address)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to touch fsvc: %v", err)
	}

	deleted, err := fsc.DeleteOld(fsvc, 0)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to delete fsvc: %v", err)
	}
	if !deleted {
		fsc.Log()
		log.Panicf("Did not delete fsvc")
	}

	_, err = fsc.GetByFunction(fsvc.Function)
	if err == nil {
		fsc.Log()
		log.Panicf("found fsvc while expecting empty cache: %v", err)
	}

	_, err = fsc.GetByFunctionUID(fsvc.Function.UID)
	if err == nil {
		fsc.Log()
		log.Panicf("found fsvc by function uid while expecting empty cache: %v", err)
	}
}
