package fscache

import (
	"fmt"
	"log"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

func TestFunctionServiceCache(t *testing.T) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := config.Build()
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
		Environment: &fv1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-env",
				UID:  "2323",
			},
			Spec: fv1.EnvironmentSpec{
				Version: 1,
				Runtime: fv1.Runtime{
					Image: "fission/foo-env",
				},
				Builder: fv1.Builder{},
			},
		},
		Address:           "xxx",
		KubernetesObjects: objects,
		Ctime:             now,
		Atime:             now,
	}
	_, err = fsc.Add(*fsvc)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to add fsvc: %v", err)
	}

	_, err = fsc.GetByFunction(fsvc.Function)
	if err != nil {
		fsc.Log()
		log.Panicf("Failed to get fsvc: %v", err)
	}
	f, err := fsc.GetByFunctionUID(fsvc.Function.UID)
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

func TestFunctionServiceNewCache(t *testing.T) {
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
		Environment: &fv1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-env",
				UID:  "2323",
			},
			Spec: fv1.EnvironmentSpec{
				Version: 1,
				Runtime: fv1.Runtime{
					Image: "fission/foo-env",
				},
				Builder: fv1.Builder{},
			},
		},
		Address:           "xxx",
		KubernetesObjects: objects,
		Ctime:             now,
		Atime:             now,
	}

	fn := &fv1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
			UID:  "1212",
		},
	}

	fsc.AddFunc(*fsvc)

	active := fsc.GetTotalAvailable(fsvc.Function)
	if active != 1 {
		logger.Panic(fmt.Sprintln("active instances not matched expected 1, found ", active))
	}

	key := fmt.Sprintf("%v_%v", fn.ObjectMeta.UID, fn.ObjectMeta.ResourceVersion)
	fsc.MarkAvailable(key, fsvc.Address)

	if fsc.GetTotalAvailable(fsvc.Function) != 0 {
		log.Panicln("active instances not matched")
	}

	_, err = fsc.GetFuncSvc(fsvc.Function)
	if err != nil {
		logger.Panic("received error while retrieving value from cache")
	}

	vals, err := fsc.ListOldForPool(30 * time.Second)
	if err != nil {
		logger.Panic("received error while get list of old values")
	}
	if len(vals) != 0 {
		logger.Panic(fmt.Sprintln("list of old values didn't matched the expected: 1", "received", len(vals)))
	}
	fsc.DeleteFunctionSvc(fsvc)
}
