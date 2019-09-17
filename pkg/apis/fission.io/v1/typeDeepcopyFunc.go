/*
Copyright 2019 The Fission Authors.

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
func (c *CanaryConfig) GetObjectKind() schema.ObjectKind {
	return &c.TypeMeta
}

func (r *Recorder) GetObjectKind() schema.ObjectKind {
	return &r.TypeMeta
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
func (c *CanaryConfig) GetObjectMeta() metav1.Object {
	return &c.Metadata
}

func (r *Recorder) GetObjectMeta() metav1.Object {
	return &r.Metadata
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
func (rl *RecorderList) GetObjectKind() schema.ObjectKind {
	return &rl.TypeMeta
}

func (cl *CanaryConfigList) GetObjectKind() schema.ObjectKind {
	return &cl.TypeMeta
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

func (rl *RecorderList) GetListMeta() metav1.ListInterface {
	return &rl.Metadata
}

func (cl *CanaryConfigList) GetListMeta() metav1.ListInterface {
	return &cl.Metadata
}

func validateMetadata(field string, m metav1.ObjectMeta) error {
	return ValidateKubeReference(field, m.Name, m.Namespace)
}

func (p *Package) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Package", p.Metadata),
		p.Spec.Validate(),
		p.Status.Validate())

	return result.ErrorOrNil()
}

func (pl *PackageList) Validate() error {
	result := &multierror.Error{}
	// not validate ListMeta
	for _, p := range pl.Items {
		result = multierror.Append(result, p.Validate())
	}
	return result.ErrorOrNil()
}

func (f *Function) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Function", f.Metadata),
		f.Spec.Validate())

	return result.ErrorOrNil()
}

func (fl *FunctionList) Validate() error {
	result := &multierror.Error{}
	for _, f := range fl.Items {
		result = multierror.Append(result, f.Validate())
	}
	return result.ErrorOrNil()
}

func (e *Environment) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Environment", e.Metadata),
		e.Spec.Validate())

	return result.ErrorOrNil()
}

func (el *EnvironmentList) Validate() error {
	result := &multierror.Error{}
	for _, e := range el.Items {
		result = multierror.Append(result, e.Validate())
	}
	return result.ErrorOrNil()
}

func (h *HTTPTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("HTTPTrigger", h.Metadata),
		h.Spec.Validate())

	return result.ErrorOrNil()
}

func (hl *HTTPTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, h := range hl.Items {
		result = multierror.Append(result, h.Validate())
	}
	return result.ErrorOrNil()
}

func (k *KubernetesWatchTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("KubernetesWatchTrigger", k.Metadata),
		k.Spec.Validate())

	return result.ErrorOrNil()
}

func (kl *KubernetesWatchTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, k := range kl.Items {
		result = multierror.Append(result, k.Validate())
	}
	return result
}

func (t *TimeTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("TimeTrigger", t.Metadata),
		t.Spec.Validate())

	return result.ErrorOrNil()
}

func (tl *TimeTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, t := range tl.Items {
		result = multierror.Append(result, t.Validate())
	}
	return result.ErrorOrNil()
}

func (m *MessageQueueTrigger) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("MessageQueueTrigger", m.Metadata),
		m.Spec.Validate())

	return result.ErrorOrNil()
}

func (ml *MessageQueueTriggerList) Validate() error {
	result := &multierror.Error{}
	for _, m := range ml.Items {
		result = multierror.Append(result, m.Validate())
	}
	return result.ErrorOrNil()
}

func (r *Recorder) Validate() error {
	result := &multierror.Error{}

	result = multierror.Append(result,
		validateMetadata("Recorder", r.Metadata),
		r.Spec.Validate())

	return result.ErrorOrNil()
}
