/*
Copyright 2020 The Kubernetes Authors.

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

package annotations

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "github.com/openshift/hypershift/api/v1alpha1/thirdparty/clusterapi/api/v1alpha4"
)

// IsPaused returns true if the Cluster is paused or the object has the `paused` annotation.
func IsPaused(cluster *clusterv1.Cluster, o metav1.Object) bool {
	if cluster.Spec.Paused {
		return true
	}
	return HasPausedAnnotation(o)
}

// HasPausedAnnotation returns true if the object has the `paused` annotation.
func HasPausedAnnotation(o metav1.Object) bool {
	return hasAnnotation(o, clusterv1.PausedAnnotation)
}

// HasSkipRemediationAnnotation returns true if the object has the `skip-remediation` annotation.
func HasSkipRemediationAnnotation(o metav1.Object) bool {
	return hasAnnotation(o, clusterv1.MachineSkipRemediationAnnotation)
}

func HasWithPrefix(prefix string, annotations map[string]string) bool {
	for key := range annotations {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// AddAnnotations sets the desired annotations on the object and returns true if the annotations have changed.
func AddAnnotations(o metav1.Object, desired map[string]string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	hasChanged := false
	for k, v := range desired {
		if cur, ok := annotations[k]; !ok || cur != v {
			annotations[k] = v
			hasChanged = true
		}
	}
	return hasChanged
}

// hasAnnotation returns true if the object has the specified annotation
func hasAnnotation(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[annotation]
	return ok
}
