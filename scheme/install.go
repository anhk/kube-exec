/*
Copyright 2017 The Kubernetes Authors.

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

package scheme

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// Register all groups in the kubectl's registry, but no componentconfig group since it's not in k8s.io/api
// The code in this file mostly duplicate the install under k8s.io/kubernetes/pkg/api and k8s.io/kubernetes/pkg/apis,
// but does NOT register the internal types.
func init() {
	// Register external types for Scheme
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(metav1beta1.AddMetaToScheme(Scheme))
	utilruntime.Must(metav1.AddMetaToScheme(Scheme))
	utilruntime.Must(scheme.AddToScheme(Scheme))
}
