/*
Copyright 2018 The Fission Authors.

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

package v1

import (
	"github.com/hashicorp/go-multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//
// To add a Fission CRD type:
//   1. Create a "spec" type, for everything in the type except metadata
//   2. Create the type with metadata + the spec
//   3. Create a list type (for example see FunctionList and Function, below)
//   4. Add methods at the bottom of this file for satisfying Object and List interfaces
//   5. Add the type to configureClient in fission/crd/client.go
//   6. Add the type to EnsureFissionCRDs in fission/crd/crd.go
//   7. Add tests to fission/crd/crd_test.go
//   8. Add a CRUD Interface type (analogous to FunctionInterface in fission/crd/function.go)
//   9. Add a getter method for your interface type to FissionClient in fission/crd/client.go
//  10. Follow the instruction in README.md to regenerate CRD type deepcopy methods
//

type (

	// Packages. Think of these as function-level images.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Package struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            PackageSpec       `json:"spec"`

		Status PackageStatus `json:"status"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	PackageList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Package `json:"items"`
	}

	// Functions.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Function struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            FunctionSpec      `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	FunctionList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Function `json:"items"`
	}

	// Environments.
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	Environment struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            EnvironmentSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	EnvironmentList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []Environment `json:"items"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	HTTPTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            HTTPTriggerSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	HTTPTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []HTTPTrigger `json:"items"`
	}

	// Kubernetes Watches as triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	KubernetesWatchTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta          `json:"metadata"`
		Spec            KubernetesWatchTriggerSpec `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	KubernetesWatchTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []KubernetesWatchTrigger `json:"items"`
	}

	// Time triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	TimeTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta `json:"metadata"`
		Spec            TimeTriggerSpec   `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	TimeTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []TimeTrigger `json:"items"`
	}

	// Message Queue triggers
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	MessageQueueTrigger struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta       `json:"metadata"`
		Spec            MessageQueueTriggerSpec `json:"spec"`
	}

	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	MessageQueueTriggerList struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ListMeta `json:"metadata"`

		Items []MessageQueueTrigger `json:"items"`
	}

	// TODO: Should this be here?
	Recorder struct {
		metav1.TypeMeta `json:",inline"`
		Metadata        metav1.ObjectMeta       `json:"metadata"`
		Spec            RecorderSpec `json:"spec"`
	}
)

// Each CRD type needs:
//   GetObjectKind (to satisfy the Object interface)
//
// In addition, each singular CRD type needs:
//   GetObjectMeta (to satisfy the ObjectMetaAccessor interface)
//
// And each list CRD type needs:
//   GetListMeta (to satisfy the ListMetaAccessor interface)

func (f *Function) GetObjectKind() schema.ObjectKind {
	return &f.TypeMeta
}
func (e *Environment) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}
func (ht *HTTPTrigger) GetObjectKind() schema.ObjectKind {
	return &ht.TypeMeta
}
func (w *KubernetesWatchTrigger) GetObjectKind() schema.ObjectKind {
	return &w.TypeMeta
}
func (t *TimeTrigger) GetObjectKind() schema.ObjectKind {
	return &t.TypeMeta
}
func (m *MessageQueueTrigger) GetObjectKind() schema.ObjectKind {
	return &m.TypeMeta
}
func (p *Package) GetObjectKind() schema.ObjectKind {
	return &p.TypeMeta
}

func (f *Function) GetObjectMeta() metav1.Object {
	return &f.Metadata
}
func (e *Environment) GetObjectMeta() metav1.Object {
	return &e.Metadata
}
func (ht *HTTPTrigger) GetObjectMeta() metav1.Object {
	return &ht.Metadata
}
func (w *KubernetesWatchTrigger) GetObjectMeta() metav1.Object {
	return &w.Metadata
}
func (t *TimeTrigger) GetObjectMeta() metav1.Object {
	return &t.Metadata
}
func (m *MessageQueueTrigger) GetObjectMeta() metav1.Object {
	return &m.Metadata
}
func (p *Package) GetObjectMeta() metav1.Object {
	return &p.Metadata
}

func (fl *FunctionList) GetObjectKind() schema.ObjectKind {
	return &fl.TypeMeta
}
func (el *EnvironmentList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}
func (hl *HTTPTriggerList) GetObjectKind() schema.ObjectKind {
	return &hl.TypeMeta
}
func (wl *KubernetesWatchTriggerList) GetObjectKind() schema.ObjectKind {
	return &wl.TypeMeta
}
func (wl *TimeTriggerList) GetObjectKind() schema.ObjectKind {
	return &wl.TypeMeta
}
func (ml *MessageQueueTriggerList) GetObjectKind() schema.ObjectKind {
	return &ml.TypeMeta
}
func (pl *PackageList) GetObjectKind() schema.ObjectKind {
	return &pl.TypeMeta
}

func (fl *FunctionList) GetListMeta() metav1.ListInterface {
	return &fl.Metadata
}
func (el *EnvironmentList) GetListMeta() metav1.ListInterface {
	return &el.Metadata
}
func (hl *HTTPTriggerList) GetListMeta() metav1.ListInterface {
	return &hl.Metadata
}
func (wl *KubernetesWatchTriggerList) GetListMeta() metav1.ListInterface {
	return &wl.Metadata
}
func (wl *TimeTriggerList) GetListMeta() metav1.ListInterface {
	return &wl.Metadata
}
func (ml *MessageQueueTriggerList) GetListMeta() metav1.ListInterface {
	return &ml.Metadata
}
func (pl *PackageList) GetListMeta() metav1.ListInterface {
	return &pl.Metadata
}

func validateMetadata(field string, m metav1.ObjectMeta) error {
	return ValidateKubeReference(field, m.Name, m.Namespace)
}

func (p *Package) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Package", p.Metadata),
		p.Spec.Validate(),
		p.Status.Validate())

	return result.ErrorOrNil()
}

func (pl *PackageList) Validate() error {
	var result *multierror.Error
	// not validate ListMeta
	for _, p := range pl.Items {
		result = multierror.Append(result, p.Validate())
	}
	return result.ErrorOrNil()
}

func (f *Function) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Function", f.Metadata),
		f.Spec.Validate())

	return result.ErrorOrNil()
}

func (fl *FunctionList) Validate() error {
	var result *multierror.Error
	for _, f := range fl.Items {
		result = multierror.Append(result, f.Validate())
	}
	return result.ErrorOrNil()
}

func (e *Environment) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Environment", e.Metadata),
		e.Spec.Validate())

	return result.ErrorOrNil()
}

func (el *EnvironmentList) Validate() error {
	var result *multierror.Error
	for _, e := range el.Items {
		result = multierror.Append(result, e.Validate())
	}
	return result.ErrorOrNil()
}

func (h *HTTPTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("HTTPTrigger", h.Metadata),
		h.Spec.Validate())

	return result.ErrorOrNil()
}

func (hl *HTTPTriggerList) Validate() error {
	var result *multierror.Error
	for _, h := range hl.Items {
		result = multierror.Append(result, h.Validate())
	}
	return result.ErrorOrNil()
}

func (k *KubernetesWatchTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("KubernetesWatchTrigger", k.Metadata),
		k.Spec.Validate())

	return result.ErrorOrNil()
}

func (kl *KubernetesWatchTriggerList) Validate() error {
	var result *multierror.Error
	for _, k := range kl.Items {
		result = multierror.Append(result, k.Validate())
	}
	return result
}

func (t *TimeTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("TimeTrigger", t.Metadata),
		t.Spec.Validate())

	return result.ErrorOrNil()
}

func (tl *TimeTriggerList) Validate() error {
	var result *multierror.Error
	for _, t := range tl.Items {
		result = multierror.Append(result, t.Validate())
	}
	return result.ErrorOrNil()
}

func (m *MessageQueueTrigger) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("MessageQueueTrigger", m.Metadata),
		m.Spec.Validate())

	return result.ErrorOrNil()
}

func (ml *MessageQueueTriggerList) Validate() error {
	var result *multierror.Error
	for _, m := range ml.Items {
		result = multierror.Append(result, m.Validate())
	}
	return result.ErrorOrNil()
}

func (r *Recorder) Validate() error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateMetadata("Recorder", r.Metadata),
		r.Spec.Validate())

	return result.ErrorOrNil()
}