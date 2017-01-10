package controller

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

type resourceContainerMap struct {
	FaultInjector *spec.FaultInjector
	Containers    []v1.Container
	ErrorValue    error
}

type resourceChangeMap struct {
	OriginalFaultInjector *spec.FaultInjector
	NewFaultInjector      *spec.FaultInjector
	ErrorValue            error
}

func TestFormatDownstreamName(t *testing.T) {
	tests := make(map[string]string)
	tests["helium"] = "faultinjector-helium"
	tests["neon"] = "faultinjector-neon"
	tests["argon"] = "faultinjector-argon"
	tests["krypton"] = "faultinjector-krypton"

	for name, expected := range tests {
		actual := formatDownstreamName(&spec.FaultInjector{ObjectMeta: v1.ObjectMeta{Name: name}})
		if actual != expected {
			t.Errorf("For FaultInjector name %v, expected %v but got %v", name, expected, actual)
		}
	}
}

func TestGenerateDownstreamContainers(t *testing.T) {
	tests := getGenerateDownstreamContainersTests()
	for name, test := range tests {
		t.Run(fmt.Sprintf("TestGenerateDownstreamContainers-%v", name), func(t *testing.T) {
			containers, err := generateDownstreamContainers(test.FaultInjector)
			if !reflect.DeepEqual(test.Containers, containers) {
				t.Errorf("Expected to receive container list %v but got container list %v", test.Containers, containers)
			}

			if err != test.ErrorValue {
				if err != nil && test.ErrorValue == nil {
					t.Errorf("Found unexpected error when generating container list: %v", err)
				} else if err == nil && test.ErrorValue != nil {
					t.Errorf("Found no error when generating container list but expected to find: %v", test.ErrorValue)
				} else if err.Error() != test.ErrorValue.Error() {
					t.Errorf("When generating container list, expected to find error:\n%v\nbut instead found:\n%v", test.ErrorValue, err)
				}
			}
		})
	}
}

func TestGenerateDownstreamLabels(t *testing.T) {
	tests := getGenerateDownstreamLabelsTests()
	for name, test := range tests {
		t.Run(fmt.Sprintf("TestGenerateDownstreamLabels-%v", name), func(t *testing.T) {
			labels := generateDownstreamLabels(test)
			if len(labels) == 0 {
				t.Error("Expected at least one label, found zero")
			} else if test.ObjectMeta.Labels == nil {
				if len(labels) != 1 {
					t.Errorf("Expected exactly one label, found %v", len(labels))
				} else {
					if labels["faultinjector-type"] != string(test.Spec.Type) {
						t.Errorf("Expected label 'faultinjector-type' to have value '%v', but got '%v'", test.Spec.Type, labels["faultinjector-type"])
					}
				}
			} else if len(labels) != len(test.ObjectMeta.Labels)+1 {
				t.Errorf("Expected %v labels, but got %v", len(test.ObjectMeta.Labels), len(labels))
			} else {
				if labels["faultinjector-type"] != string(test.Spec.Type) {
					t.Errorf("Expected label 'faultinjector-type' to have value '%v', but got '%v'", test.Spec.Type, labels["faultinjector-type"])
				}
				for label, value := range test.ObjectMeta.Labels {
					if labels[label] != value {
						t.Errorf("Expected label '%v' to have value '%v', but got '%v'", label, value, labels[label])
					}
				}
			}
		})
	}
}

func TestGenerateDownstreamObject(t *testing.T) {
	tests := getGenerateDownstreamObjectTests()
	for name, test := range tests {
		var expectedObj *extensionsobj.Deployment
		containers, expectedErr := generateDownstreamContainers(test)
		if expectedErr != nil {
			expectedObj = nil
		} else {
			expectedObj = &extensionsobj.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      formatDownstreamName(test),
					Namespace: v1.NamespaceDefault,
					Labels:    map[string]string{"generatedBy": "FaultInjector"},
				},
				Spec: extensionsobj.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: generateDownstreamLabels(test),
						},
						Spec: v1.PodSpec{
							Containers: containers,
						},
					},
				},
			}
		}
		t.Run(name, func(t *testing.T) {
			actualObj, actualErr := generateDownstreamObject(test)
			if !reflect.DeepEqual(expectedObj, actualObj) {
				t.Errorf("Expected to retrieve downstream obj:\n%v\nbut got\n%v", expectedObj, actualObj)
			}
			if actualErr != expectedErr {
				if actualErr != nil && expectedErr == nil {
					t.Errorf("Found unexpected error when generating downstream object: %v", actualErr)
				} else if actualErr == nil && expectedErr != nil {
					t.Errorf("Found no error when generating downstream object but expected to find: %v", expectedErr)
				} else if actualErr.Error() != expectedErr.Error() {
					t.Errorf("When generating downstream object, expected to find error:\n%v\nbut instead found:\n%v", expectedErr, actualErr)
				}
			}
		})
	}
}

func TestUpdateDownstreamObject(t *testing.T) {
	tests := getUpdateDownstreamObjectTests()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actualObj, err := generateDownstreamObject(test.OriginalFaultInjector)
			if err != nil {
				t.Fatalf("Found unexpected exception when generating initial object: %v", err)
			}
			originalObj, err := generateDownstreamObject(test.OriginalFaultInjector)
			actualErr := updateDownstreamObject(actualObj, test.NewFaultInjector)
			if test.ErrorValue == nil {
				expectedObj, err := generateDownstreamObject(test.NewFaultInjector)
				if err != nil {
					t.Fatalf("Found unexpected exception when generating expected object: %v", err)
				}
				if actualErr != nil {
					t.Errorf("Found unexpected error when generating expected object: %v", err)
				}
				if !reflect.DeepEqual(expectedObj, actualObj) {
					t.Errorf("Expected resultant object:\n%v\nbut got\n%v", expectedObj, actualObj)
				}
			} else {
				if !reflect.DeepEqual(originalObj, actualObj) {
					t.Errorf("Unexpected change made to object. Expected:\n%v\nbut got\n%v", originalObj, actualObj)
				}
				if actualErr != test.ErrorValue {
					if actualErr != nil && test.ErrorValue == nil {
						t.Errorf("Found unexpected error when updating downstream object: %v", actualErr)
					} else if actualErr == nil && test.ErrorValue != nil {
						t.Errorf("Found no error when updating downstream object but expected to find: %v", test.ErrorValue)
					} else if actualErr.Error() != test.ErrorValue.Error() {
						t.Errorf("When updating downstream object, expected to find error:\n%v\nbut instead found:\n%v", test.ErrorValue, actualErr)
					}
				}
			}
		})
	}
}

func getGenerateDownstreamContainersTests() map[string]resourceContainerMap {
	tests := make(map[string]resourceContainerMap)
	tests["PodKiller"] = resourceContainerMap{
		FaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "hydrogen",
				Namespace: v1.NamespaceDefault,
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		Containers: []v1.Container{
			{Name: "fault-injector-podkiller", Image: fmt.Sprintf("%v/fault-injector-podkiller:%v", imagePrefix, version.Version)},
		},
		ErrorValue: nil,
	}
	tests["NetworkLatency"] = resourceContainerMap{
		FaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "helium",
				Namespace: v1.NamespaceDefault,
			},
			Spec: spec.FaultInjectorSpec{
				Type: "NetworkLatency",
			},
		},
		Containers: nil,
		ErrorValue: fmt.Errorf("Unsupported value NetworkLatency for spec.type on the FaultInjector"),
	}
	return tests
}

func getGenerateDownstreamLabelsTests() map[string]*spec.FaultInjector {
	tests := make(map[string]*spec.FaultInjector)
	tests["NilLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "lithium",
			Namespace: v1.NamespaceDefault,
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	tests["EmptyLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "sodium",
			Namespace: v1.NamespaceDefault,
			Labels:    make(map[string]string),
		},
		Spec: spec.FaultInjectorSpec{
			Type: "NetworkLatency",
		},
	}
	tests["OneLabel"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "potassium",
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"period": "four",
			},
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	tests["MultipleLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "rubidium",
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"group":  "alkali",
				"period": "four",
			},
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	return tests
}

func getGenerateDownstreamObjectTests() map[string]*spec.FaultInjector {
	tests := make(map[string]*spec.FaultInjector)
	tests["PodKiller-NilLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "lithium",
			Namespace: v1.NamespaceDefault,
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	tests["NetworkLatency-EmptyLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "sodium",
			Namespace: v1.NamespaceDefault,
			Labels:    make(map[string]string),
		},
		Spec: spec.FaultInjectorSpec{
			Type: "NetworkLatency",
		},
	}
	tests["PodKiller-OneLabel"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "potassium",
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"period": "four",
			},
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	tests["PodKiller-MultipleLabels"] = &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "rubidium",
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"group":  "alkali",
				"period": "four",
			},
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	return tests
}

func getUpdateDownstreamObjectTests() map[string]resourceChangeMap {
	tests := make(map[string]resourceChangeMap)
	tests["NoChange"] = resourceChangeMap{
		OriginalFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "hydrogen",
				Namespace: v1.NamespaceDefault,
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		NewFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "hydrogen",
				Namespace: v1.NamespaceDefault,
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		ErrorValue: nil,
	}
	tests["AddLabel"] = resourceChangeMap{
		OriginalFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "lithium",
				Namespace: v1.NamespaceDefault,
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		NewFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "lithium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkali",
					"period": "two",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		ErrorValue: nil,
	}
	tests["RemoveLabel"] = resourceChangeMap{
		OriginalFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "sodium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"atomicNumber": "11",
					"group":        "alkali",
					"period":       "three",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		NewFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "sodium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkali",
					"period": "three",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		ErrorValue: nil,
	}
	tests["DifferentName"] = resourceChangeMap{
		OriginalFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "potassium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkali",
					"period": "four",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		NewFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "calcium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkaline-earth",
					"period": "four",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		ErrorValue: fmt.Errorf("Expected downstream object to have the same name as upstream object (%v), but got %v",
			formatDownstreamName(&spec.FaultInjector{ObjectMeta: v1.ObjectMeta{Name: "calcium"}}),
			formatDownstreamName(&spec.FaultInjector{ObjectMeta: v1.ObjectMeta{Name: "potassium"}})),
	}
	tests["InvalidType"] = resourceChangeMap{
		OriginalFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "rubidium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkali",
					"period": "four",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "PodKiller",
			},
		},
		NewFaultInjector: &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      "rubidium",
				Namespace: v1.NamespaceDefault,
				Labels: map[string]string{
					"group":  "alkali",
					"period": "five",
				},
			},
			Spec: spec.FaultInjectorSpec{
				Type: "NetworkLatency",
			},
		},
		ErrorValue: fmt.Errorf("Unsupported value NetworkLatency for spec.type on the FaultInjector"),
	}
	return tests
}
