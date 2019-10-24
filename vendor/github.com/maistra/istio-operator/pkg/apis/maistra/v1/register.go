// NOTE: Boilerplate only.  Ignore this file.

// Package v1 contains API Schema definitions for the maistra v1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=maistra.io
package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "maistra.io", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

var (
	GroupName     = SchemeGroupVersion.Group
	GroupVersion  = schema.GroupVersion{Group: GroupName, Version: "v1"}
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, corev1.AddToScheme)
	// Install is a function which adds this version to a scheme
	Install = schemeBuilder.AddToScheme

	// AddToScheme exists solely to keep the old generators creating valid code
	// DEPRECATED
	AddToScheme = schemeBuilder.AddToScheme
)

// Resource generated code relies on this being here, but it logically belongs to the group
// DEPRECATED
func Resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: GroupName, Resource: resource}
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&ServiceMeshMemberRoll{},
		&ServiceMeshMemberRollList{},
		&ServiceMeshControlPlane{},
		&ServiceMeshControlPlaneList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
