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

package crd

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheKey : Given metadata, create a key that uniquely identifies the contents
// of the object. Since resourceVersion changes on every update and
// UIDs are unique, uid+resourceVersion identifies the
// content. (ResourceVersion may also update on status updates, so
// this will result in some unnecessary cache misses. That should be
// ok.)
func CacheKey(metadata *metav1.ObjectMeta) string {
	return fmt.Sprintf("%v_%v", metadata.UID, metadata.ResourceVersion)
}
