package spec

import (
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

// FaultInjector defines a FaultInjector deployment.
type FaultInjector struct {
	unversioned.TypeMeta `json:",inline"`
	v1.ObjectMeta        `json:"metadata,omitempty"`
	Spec                 FaultInjectorSpec `json:"spec"`
}

// FaultInjectorList is a list of FaultInjectors.
type FaultInjectorList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []*FaultInjector `json:"items"`
}

// FaultInjectorSpec holds specification parameters for a FaultInjector deployment.
type FaultInjectorSpec struct {
}
