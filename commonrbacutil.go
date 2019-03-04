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

package fission

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
)

// This file has util functions needed for setting up and cleaning up RBAC objects.

const (
	MaxRetries = 10
)

// MakeSAObj returns a ServiceAccount object with the given SA name and namespace
func MakeSAObj(sa, ns string) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      sa,
		},
	}
}

// SetupSA checks if a service account is present in the namespace, if not creates it.
func SetupSA(k8sClient *kubernetes.Clientset, sa, ns string) (*apiv1.ServiceAccount, error) {
	saObj, err := k8sClient.CoreV1().ServiceAccounts(ns).Get(sa, metav1.GetOptions{})
	if err == nil {
		return saObj, nil
	}

	if k8serrors.IsNotFound(err) {
		saObj = MakeSAObj(sa, ns)
		saObj, err = k8sClient.CoreV1().ServiceAccounts(ns).Create(saObj)
	}

	return saObj, err
}

// makeRoleBindingObj is a helper function called from other functions in this file only.
// given a rolebinging name and namespace, it makes a rolebinding object mapping the role to the SA of the namespace.
func makeRoleBindingObj(roleBinding, roleBindingNs, role, roleKind, sa, saNamespace string) *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBinding,
			Namespace: roleBindingNs,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: saNamespace,
			},
		},
		RoleRef: rbac.RoleRef{
			Kind: roleKind,
			Name: role,
		},
	}
}

// isSAInRoleBinding checkis if a service account is present in the rolebinding object
func isSAInRoleBinding(rbObj *rbac.RoleBinding, sa, ns string) bool {
	for _, subject := range rbObj.Subjects {
		if subject.Name == sa && subject.Namespace == ns {
			return true
		}
	}

	return false
}

// PatchSpec is a standard struct needed for JSON merge.
type PatchSpec struct {
	Op    string       `json:"op"`
	Path  string       `json:"path"`
	Value rbac.Subject `json:"value"`
}

// AddSaToRoleBindingWithRetries adds a service account to a rolebinding object. IT retries on already exists and conflict errors.
func AddSaToRoleBindingWithRetries(logger *zap.Logger, k8sClient *kubernetes.Clientset, roleBinding, roleBindingNs, sa, saNamespace, role, roleKind string) (err error) {
	patch := PatchSpec{}
	patch.Op = "add"
	patch.Path = "/subjects/-"
	patch.Value = rbac.Subject{
		Kind:      "ServiceAccount",
		Name:      sa,
		Namespace: saNamespace,
	}

	patchJson, err := json.Marshal([]PatchSpec{patch})
	if err != nil {
		logger.Error("error marshalling patch into json", zap.Error(err))
		return err
	}

	for i := 0; i < MaxRetries; i++ {
		_, err = k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Patch(roleBinding, types.JSONPatchType, patchJson)
		if err == nil {
			logger.Info("patched rolebinding",
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			return err
		}

		if k8serrors.IsNotFound(err) {
			logger.Info("rolebinding not found - will try to create it",
				zap.Error(err),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			// someone may have deleted the object between us checking if the object is present and deciding to patch
			// so just create the object again
			rbObj := makeRoleBindingObj(roleBinding, roleBindingNs, role, roleKind, sa, saNamespace)
			rbObj, err = k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Create(rbObj)
			if err == nil {
				logger.Info("created rolebinding",
					zap.String("role_binding", roleBinding),
					zap.String("role_binding_namespace", roleBindingNs))
				return err
			}

			if k8serrors.IsAlreadyExists(err) {
				logger.Info("rolebinding object already exists, retrying patch",
					zap.Error(err),
					zap.String("role_binding", roleBinding),
					zap.String("role_binding_namespace", roleBindingNs))
				continue
			}

			return errors.Wrap(err, "error returned by rolebinding create")
		}

		if k8serrors.IsConflict(err) {
			// TODO : Need to test this out, not able to simulate conflicts yet.
			// Initially, my understanding was that patch can never error on conflict because Api server will handle the conflicts for us.
			// but one CI run did show patch errored out on conflict : https://api.travis-ci.org/v3/job/373161490/log.txt, look for :
			// Error returned by rolebinding patch : <some more text> there is a meaningful conflict (firstResourceVersion: "35482724", currentResourceVersion: "35482849")
			// so, m guessing retrying patch should help. will watch out for any such conflicts and fix the issue if any
			logger.Info("conflict reported on patch of rolebinding - retrying patch operation",
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			continue
		}

		return errors.Wrap(err, "error returned by rolebinding patch")
	}

	return errors.Wrapf(err, "exceeded max retries (%d) adding SA: %s.%s to rolebinding: %s.%s, giving up", MaxRetries, sa, saNamespace, roleBinding, roleBindingNs)
}

// RemoveSAFromRoleBindingWithRetries removes an SA from the rolebinding passed as parameter. If this is the only SA in
// the rolebinding, then it deletes the rolebinding object.
func RemoveSAFromRoleBindingWithRetries(logger *zap.Logger, k8sClient *kubernetes.Clientset, roleBinding, roleBindingNs string, saToRemove map[string]bool) (err error) {
	for i := 0; i < MaxRetries; i++ {
		rbObj, err := k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Get(
			roleBinding, metav1.GetOptions{})
		if err != nil {
			// silently ignoring the error. there's no need for us to remove sa anymore.
			logger.Info("rolebinding not found, but ignoring the error since we're cleaning up",
				zap.Error(err),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			return nil
		}

		subjects := rbObj.Subjects
		newSubjects := make([]rbac.Subject, 0)

		// TODO : optimize it.
		for _, item := range subjects {
			if _, ok := saToRemove[MakeSAMapKey(item.Name, item.Namespace)]; ok {
				continue
			}

			newSubjects = append(newSubjects, rbac.Subject{
				Kind:      "ServiceAccount",
				Name:      item.Name,
				Namespace: item.Namespace,
			})
		}
		if len(newSubjects) == 0 {
			return DeleteRoleBinding(k8sClient, roleBinding, roleBindingNs)
		}

		rbObj.Subjects = newSubjects

		// cant use patch for deletes, the results become in-deterministic, so using update.
		_, err = k8sClient.RbacV1beta1().RoleBindings(rbObj.Namespace).Update(rbObj)
		switch {
		case err == nil:
			logger.Info("removed service accounts from rolebinding",
				zap.Any("service_accounts", saToRemove),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			return nil
		case k8serrors.IsConflict(err):
			logger.Info("conflict in update of rolebinding - retrying",
				zap.Error(err),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			continue
		default:
			return errors.Wrap(err, "rolebinding update errored out")
		}
	}

	return errors.Wrapf(err, "max retries: %d exceeded for removing SA's: %v from rolebinding %s.%s, giving up", MaxRetries, saToRemove, roleBinding, roleBindingNs)
}

// SetupRoleBinding adds a role to a service account if the rolebinding object is already present in the namespace.
// if not, it creates a rolebinding object granting the role to the SA in the namespace.
func SetupRoleBinding(logger *zap.Logger, k8sClient *kubernetes.Clientset, roleBinding, roleBindingNs, role, roleKind, sa, saNamespace string) error {
	// get the role binding object
	rbObj, err := k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Get(
		roleBinding, metav1.GetOptions{})

	if err == nil {
		if !isSAInRoleBinding(rbObj, sa, saNamespace) {
			logger.Info("service account is not present in the rolebinding - will add",
				zap.String("service_account_name", sa),
				zap.String("service_account_namespace", saNamespace),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			return AddSaToRoleBindingWithRetries(logger, k8sClient, roleBinding, roleBindingNs, sa, saNamespace, role, roleKind)
		}
		logger.Info("service account already present in rolebinding so nothing to add",
			zap.String("service_account_name", sa),
			zap.String("service_account_namespace", saNamespace),
			zap.String("role_binding", roleBinding),
			zap.String("role_binding_namespace", roleBindingNs))
		return nil
	}

	// if role binding is missing, create it. also add this sa to the binding.
	if k8serrors.IsNotFound(err) {
		logger.Info("rolebinding does NOT exist in namespace - creating it",
			zap.Error(err),
			zap.String("role_binding", roleBinding),
			zap.String("role_binding_namespace", roleBindingNs))
		rbObj = makeRoleBindingObj(roleBinding, roleBindingNs, role, roleKind, sa, saNamespace)
		rbObj, err = k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Create(rbObj)
		if k8serrors.IsAlreadyExists(err) {
			logger.Info("rolebinding already exists in namespace - adding service account to rolebinding",
				zap.String("service_account_name", sa),
				zap.String("service_account_namespace", saNamespace),
				zap.String("role_binding", roleBinding),
				zap.String("role_binding_namespace", roleBindingNs))
			err = AddSaToRoleBindingWithRetries(logger, k8sClient, roleBinding, roleBindingNs, sa, saNamespace, role, roleKind)
		}
	}

	return err
}

// DeleteRoleBinding deletes a rolebinding object. if k8s throws an error that the rolebinding is not there, it just
// returns silently.
func DeleteRoleBinding(k8sClient *kubernetes.Clientset, roleBinding, roleBindingNs string) error {
	// if deleteRoleBinding is invoked by 2 fission services at the same time for the same rolebinding,
	// the first call will succeed while the 2nd will fail with isNotFound. but we dont want to error out then.
	err := k8sClient.RbacV1beta1().RoleBindings(roleBindingNs).Delete(roleBinding, &metav1.DeleteOptions{})
	if err == nil || k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func MakeSAMapKey(saName, saNamespace string) string {
	return fmt.Sprintf("%s-%s", saName, saNamespace)
}
