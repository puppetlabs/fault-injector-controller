package controller

import (
	"testing"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	fkubernetes "k8s.io/client-go/1.5/kubernetes/fake"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	ktesting "k8s.io/client-go/1.5/testing"
	"k8s.io/client-go/1.5/tools/cache"
	fcache "k8s.io/client-go/1.5/tools/cache/testing"
)

type resourceEvent struct {
	eventType    string
	resourceName string
	resourceSpec spec.FaultInjectorSpec
	result       ktesting.Action
}

type resourceEventTest struct {
	eventList []resourceEvent
	source    *fcache.FakeControllerSource
	dest      *fkubernetes.Clientset
}

func TestCreateTPR(t *testing.T) {
	expectedResource := unversioned.GroupVersionResource{
		Group:    "extensions",
		Version:  "v1beta1",
		Resource: "thirdpartyresources",
	}

	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset()
	c := &FaultInjectorController{
		kclient: clientset,
	}
	c.createTPR()

	actions := clientset.Fake.Actions()
	firstAction, actions := actions[0], actions[1:]
	if firstAction.GetVerb() != "create" {
		t.Error("Expected to create the FaultInjector ThirdPartyResource first, but did not.")
	} else if res := firstAction.GetResource(); res != expectedResource {
		t.Error("Expected to create a ThirdPartyResource but created", res)
	} else if tpr := firstAction.(ktesting.CreateAction).GetObject().(*extensionsobj.ThirdPartyResource); tpr.ObjectMeta.Name != "fault-injector.k8s.puppet.com" {
		t.Error("Expected to create a ThirdPartyResource for fault-injector.k8s.puppet.com, but created", tpr.ObjectMeta.Name)
	} else if tpr := firstAction.(ktesting.CreateAction).GetObject().(*extensionsobj.ThirdPartyResource); len(tpr.Versions) != 1 {
		t.Error("Expected FaultInjector ThirdPartyResource to have exactly one version but got", len(tpr.Versions))
	} else if tpr := firstAction.(ktesting.CreateAction).GetObject().(*extensionsobj.ThirdPartyResource); tpr.Versions[0].Name != version.ResourceAPIVersion {
		t.Error("Expected FaultInjector ThirdPartyResource to be version", version.ResourceAPIVersion, "but got", tpr.Versions[0].Name)
	}
}

func TestResourceHandlerFuncs(t *testing.T) {
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset()
	c := &FaultInjectorController{
		kclient: clientset,
	}

	resourceHandlers := &cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFaultInjector,
		DeleteFunc: c.handleDeleteFaultInjector,
		UpdateFunc: c.handleUpdateFaultInjector,
	}

	source := fcache.NewFakeControllerSource()

	_, controller := cache.NewInformer(
		source,
		&spec.FaultInjector{},
		time.Millisecond*100,
		resourceHandlers)

	stop := make(chan struct{})
	defer close(stop)

	go controller.Run(stop)

	addTests := &resourceEventTest{
		eventList: []resourceEvent{
			{"add", "helium", spec.FaultInjectorSpec{}, ktesting.CreateActionImpl{}},
		},
		source: source,
		dest:   clientset,
	}

	modifyTests := &resourceEventTest{
		eventList: []resourceEvent{
			{"modify", "lithium", spec.FaultInjectorSpec{}, ktesting.UpdateActionImpl{}},
		},
		source: source,
		dest:   clientset,
	}

	deleteTests := &resourceEventTest{
		eventList: []resourceEvent{
			{"delete", "fluorine", spec.FaultInjectorSpec{}, ktesting.DeleteActionImpl{}},
		},
		source: source,
		dest:   clientset,
	}

	t.Run("addTests", addTests.testEvents)
	t.Run("modifyTests", modifyTests.testEvents)
	t.Run("deleteTests", deleteTests.testEvents)
}

func (e *resourceEventTest) testEvents(t *testing.T) {
	for _, event := range e.eventList {
		switch event.eventType {
		case "add":
			obj := &spec.FaultInjector{
				ObjectMeta: v1.ObjectMeta{Name: event.resourceName},
				Spec:       event.resourceSpec,
			}
			e.source.Add(obj)
		case "modify":
			obj := &spec.FaultInjector{
				ObjectMeta: v1.ObjectMeta{Name: event.resourceName},
			}
			e.source.Add(obj)
			obj.Spec = event.resourceSpec
			e.source.Modify(obj)
		case "delete":
			obj := &spec.FaultInjector{
				ObjectMeta: v1.ObjectMeta{Name: event.resourceName},
				Spec:       event.resourceSpec,
			}
			e.source.Add(obj)
			e.source.Delete(obj)
		}
		actions := e.dest.Fake.Actions()
		success := false
		for _, a := range actions {
			if a == event.result {
				success = true
				break
			}
		}
		if !success {
			t.Error("Expected", event.eventType, "action for", event.resourceName,
				"to result in a", event.result.GetVerb(), "action on", event.result.GetResource().Resource,
				"but that did not happen.")
		}
	}
}
