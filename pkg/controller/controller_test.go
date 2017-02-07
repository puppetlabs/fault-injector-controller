package controller

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	"strings"

	fkubernetes "k8s.io/client-go/1.5/kubernetes/fake"
	"k8s.io/client-go/1.5/pkg/api"
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

func TestPrepareInitialStore(t *testing.T) {
	c, _ := prepareResourceHandlerTest()
	clientset := c.kclient
	store := c.store

	sources, err := generateTestFaultInjectors(3)
	if err != nil {
		t.Fatalf("Found unexpected error when generating test cases: %v", err)
	}
	for i := range sources {
		deployment, err := generateDownstreamObject(sources[i])
		if err != nil {
			t.Fatalf("Found unexpected error when preparing test cases: %v", err)
		}
		clientset.Extensions().Deployments(v1.NamespaceDefault).Create(deployment)
	}
	otherDeployment := &extensionsobj.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "NotFaultInjector",
			Namespace: v1.NamespaceDefault,
			Labels:    map[string]string{"generatedBy": "foobar"},
		},
		Spec: extensionsobj.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Name: "foo", Image: "gcr.io/google-samples/gb-frontend:v4"},
					},
				},
			},
		},
	}
	clientset.Extensions().Deployments(v1.NamespaceDefault).Create(otherDeployment)

	err = c.prepareInitialStore()
	if err != nil {
		t.Fatalf("Found unexpected error when preparing initial store: %v", err)
	}

	rawActualResources := store.List()
	actualDeployments := make([]*extensionsobj.Deployment, len(rawActualResources))
	for i := range rawActualResources {
		resource := rawActualResources[i].(*spec.FaultInjector)
		deployment, err := generateDownstreamObject(resource)
		if err != nil {
			t.Errorf("When generating deployment from object in store:\n%v\nfound unexpected error:\n%v\n", resource, err)
		} else {
			actualDeployments[i] = deployment
		}
	}
	validateResourceList(t, actualDeployments, sources)
}

func TestGetDownstreamState(t *testing.T) {
	c, _ := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)
	defaultSource := &spec.FaultInjector{
		ObjectMeta: v1.ObjectMeta{
			Name:      "hydrogen",
			Namespace: v1.NamespaceDefault,
		},
		Spec: spec.FaultInjectorSpec{
			Type: "PodKiller",
		},
	}
	defaultDeployment, err := generateDownstreamObject(defaultSource)
	if err != nil {
		t.Fatalf("Found unexpected exception when generating initial downstream object: %v", err)
	}
	namespaceSourceList, err := generateTestFaultInjectors(1)
	namespaceSource := namespaceSourceList[0]
	namespaceDeployment, err := generateDownstreamObject(namespaceSource)
	if err != nil {
		t.Fatalf("Found unexpected exception when generating initial downstream object: %v", err)
	}

	testResponse := func(source *spec.FaultInjector) func(t *testing.T) {
		return func(t *testing.T) {
			expected, err := clientset.Extensions().Deployments(source.ObjectMeta.Namespace).Get(formatDownstreamName(source))
			actual := c.getDownstreamState(source)
			if err == nil {
				if !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected to find:\n%v\nbut instead found:\n%v", expected, actual)
				}
			} else {
				if actual != nil {
					t.Errorf("Found unexpected object when expected nil:\n%v", actual)
				}
			}
		}
	}

	t.Run("MissingObject", testResponse(defaultSource))

	clientset.Extensions().Deployments(v1.NamespaceDefault).Create(defaultDeployment)
	t.Run("ObjectExistsDefaultNamespace", testResponse(defaultSource))

	clientset.Extensions().Deployments(namespaceDeployment.ObjectMeta.Namespace).Create(namespaceDeployment)
	t.Run("ObjectExistsDifferentNamespace", testResponse(namespaceSource))
}

func TestAddFaultInjector(t *testing.T) {
	c, _ := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)

	sources, err := generateTestFaultInjectors(3)
	if err != nil {
		t.Fatalf("Error when generating resources for test: %v", err)
	}

	deployments := getDeploymentList(clientset, t)
	if len(deployments) != 0 {
		t.Fatalf("Found unexpected deployment resources: %v", deployments)
	}

	t.Run("InitialAdd", func(t *testing.T) {
		err = c.addFaultInjector(sources[0])
		if err != nil {
			t.Errorf("Found unexpected error when adding resource: %v", err)
		}
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[0:1])
	})

	t.Run("ReAdd", func(t *testing.T) {
		err = c.addFaultInjector(sources[0])
		if err != nil {
			t.Errorf("Found unexpected error when adding resource: %v", err)
		}
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[0:1])
	})

	t.Run("SecondAdd", func(t *testing.T) {
		err = c.addFaultInjector(sources[1])
		if err != nil {
			t.Errorf("Found unexpected error when adding resource: %v", err)
		}
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[:2])
	})

	t.Run("Update", func(t *testing.T) {
		sources[2].ObjectMeta.Name = sources[0].ObjectMeta.Name
		err = c.addFaultInjector(sources[2])
		if err != nil {
			t.Errorf("Found unexpected error when adding resource: %v", err)
		}
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[1:])
	})

	t.Run("AddDifferentNamespace", func(t *testing.T) {
		var newResource spec.FaultInjector
		newResource = *sources[1]
		newResource.ObjectMeta.Namespace = v1.NamespaceDefault
		sources = append(sources, &newResource)
		err = c.addFaultInjector(&newResource)
		if err != nil {
			t.Errorf("Found unexpected error when adding resource: %v", err)
		}
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[1:])
	})
}

func TestDeleteFaultInjector(t *testing.T) {
	count := 2

	c, _ := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)
	sources, err := generateTestFaultInjectors(count)
	if err != nil {
		t.Fatalf("Error when generating resources for test: %v", err)
	}
	for _, source := range sources {
		deployment, err := generateDownstreamObject(source)
		if err != nil {
			t.Fatalf("Unexpected exception found when generating initial objects: %v", err)
		}
		clientset.Extensions().Deployments(v1.NamespaceAll).Create(deployment)
	}
	deployments := getDeploymentList(clientset, t)
	if len(deployments) != count {
		t.Fatalf("Expected to retrieve %v deployments but found %v when preparing for test:\n%v", count, len(deployments), deployments)
	}
	err = c.deleteFaultInjector(sources[0])
	if err != nil {
		t.Errorf("Found unexpected error when deleting resource: %v", err)
	}
	newDeployments := getDeploymentList(clientset, t)
	if len(newDeployments) != count-1 {
		t.Errorf("Expected to find %v deployments after deletion, but found %v", count-1, len(newDeployments))
	}
	if !reflect.DeepEqual(deployments[1:], newDeployments) {
		t.Errorf("Expected deployments:\n%v\nbut got\n%v", deployments[1:], newDeployments)
	}
}

func TestAddResourceHandlerFuncs(t *testing.T) {
	c, source := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)
	controller := c.controller

	stop := make(chan struct{})
	defer close(stop)

	go controller.Run(stop)

	sources, err := generateTestFaultInjectors(3)
	if err != nil {
		t.Fatalf("Error when generating resources for test: %v", err)
	}

	deployments := getDeploymentList(clientset, t)
	if len(deployments) != 0 {
		t.Fatalf("Found unexpected deployment resources: %v", deployments)
	}

	t.Run("InitialAdd", func(t *testing.T) {
		source.Add(sources[0])
		time.Sleep(time.Second)
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[0:1])
	})

	t.Run("ReAdd", func(t *testing.T) {
		source.Add(sources[0])
		time.Sleep(time.Second)
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[0:1])
	})

	t.Run("SecondAdd", func(t *testing.T) {
		source.Add(sources[1])
		time.Sleep(time.Second)
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[:2])
	})

	t.Run("Update", func(t *testing.T) {
		sources[2].ObjectMeta.Name = sources[0].ObjectMeta.Name
		source.Add(sources[2])
		time.Sleep(time.Second)
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[1:])
	})
}

func TestDeleteResourceHandlerFunc(t *testing.T) {
	c, source := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)
	controller := c.controller

	stop := make(chan struct{})
	defer close(stop)

	go controller.Run(stop)

	sources, err := generateTestFaultInjectors(3)
	if err != nil {
		t.Fatalf("Error when generating resources for test: %v", err)
	}

	deployments := getDeploymentList(clientset, t)
	if len(deployments) != 0 {
		t.Fatalf("Found unexpected deployment resources: %v", deployments)
	}

	t.Run("InitialAdd", func(t *testing.T) {
		for j := range sources {
			source.Add(sources[j])
		}
		time.Sleep(time.Second * 5)
		deployments = getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources)
	})

	upstreamObjects := make([]*spec.FaultInjector, len(sources))
	rawListObj, err := source.List(api.ListOptions{})
	if err != nil {
		t.Fatalf("Found unexpected error when preparing resources for test: %v", err)
	}
	rawList := rawListObj.(*api.List).Items
	list := make([]*spec.FaultInjector, len(rawList))
	for i := range rawList {
		list[i] = rawList[i].(*spec.FaultInjector)
	}
	for j := range sources {
		for i := range list {
			if list[i].ObjectMeta.Name == sources[j].ObjectMeta.Name {
				upstreamObjects[j] = list[i]
			}
		}
	}

	t.Run("FirstDelete", func(t *testing.T) {
		source.Delete(upstreamObjects[2])
		time.Sleep(time.Second)
		deployments := getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[:2])
	})

	t.Run("SecondDelete", func(t *testing.T) {
		source.Delete(upstreamObjects[1])
		time.Sleep(time.Second)
		deployments := getDeploymentList(clientset, t)
		validateResourceList(t, deployments, sources[:1])
	})
}

func TestControllerRun(t *testing.T) {
	c, source := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)

	sources, err := generateTestFaultInjectors(5)
	if err != nil {
		t.Fatalf("Error when generating resources for test: %v", err)
	}

	// Set up initial state:
	// * source[0] is to be removed.
	// * source[1] should be changed to match source[4].
	// * source[2] should stay as-is.
	// * source[3] should be added.
	for i := range sources[:3] {
		err = c.addFaultInjector(sources[i])
		if err != nil {
			t.Fatalf("Found unexpected error when adding initial resource: %v", err)
		}
	}

	deployments := getDeploymentList(clientset, t)
	if len(deployments) != 3 {
		t.Fatalf("Expected 3 starting deployment resources, but found %v", len(deployments))
	}

	sources[4].ObjectMeta.Name = sources[1].ObjectMeta.Name
	for _, resource := range sources[2:] {
		source.Add(resource)
	}
	source.AddDropWatch(sources[0])
	rawList, err := source.List(api.ListOptions{})
	if err != nil {
		t.Fatalf("Found unexpected error when preparing resources for test: %v", err)
	}
	list := rawList.(*api.List).Items
	var sourceToDelete *spec.FaultInjector
	for i := range list {
		resource := list[i].(*spec.FaultInjector)
		if resource.ObjectMeta.Name == sources[0].ObjectMeta.Name {
			sourceToDelete = resource
			break
		}
	}
	source.Delete(sourceToDelete)

	stop := make(chan struct{})
	defer close(stop)

	go c.Run(stop)
	time.Sleep(time.Second * 5)
	deployments = getDeploymentList(clientset, t)
	t.Run("DeploymentCount", func(t *testing.T) {
		if len(deployments) != 3 {
			stringDeployments := make([]string, len(deployments))
			for i := range deployments {
				stringDeployments[i] = fmt.Sprintf("%#v", deployments[i])
			}
			t.Errorf("Expected 3 final deployments, but found %v:\n%v", len(deployments), strings.Join(stringDeployments, "\n"))
		}
	})

	t.Run("Deleted", func(t *testing.T) {
		validateSingleResourceAbsent(t, deployments, sources[0])
	})

	t.Run("Unchanged", func(t *testing.T) {
		validateSingleResourcePresent(t, deployments, sources[2])
	})

	t.Run("Changed", func(t *testing.T) {
		validateSingleResourcePresent(t, deployments, sources[4])
		validateSingleResourceAbsent(t, deployments, sources[1])
	})

	t.Run("Added", func(t *testing.T) {
		validateSingleResourcePresent(t, deployments, sources[3])
	})

	t.Run("DownstreamMissing", func(t *testing.T) {
		err = c.deleteFaultInjector(sources[2])
		if err != nil {
			t.Fatalf("Error when preparing resources for test: %v", err)
		}
		time.Sleep(time.Second)
		deployments = getDeploymentList(clientset, t)
		validateSingleResourcePresent(t, deployments, sources[2])
	})
}

func TestCreateTPR(t *testing.T) {
	expectedResource := unversioned.GroupVersionResource{
		Group:    "extensions",
		Version:  "v1beta1",
		Resource: "thirdpartyresources",
	}

	c, _ := prepareResourceHandlerTest()
	clientset := c.kclient.(*fkubernetes.Clientset)
	c.createTPR()

	actions := clientset.Fake.Actions()
	// We create two namespaces in prepareResourceHandlerTest(): skip those.
	firstAction, actions := actions[2], actions[3:]
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

func getDeploymentList(clientset *fkubernetes.Clientset, t *testing.T) []*extensionsobj.Deployment {
	var out []*extensionsobj.Deployment
	deployments, err := clientset.Extensions().Deployments(v1.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		t.Fatalf("Error when retrieving initial downstream objects for test: %v", err)
		return make([]*extensionsobj.Deployment, 0)
	}
	out = make([]*extensionsobj.Deployment, len(deployments.Items))
	for i := range deployments.Items {
		out[i] = &deployments.Items[i]
	}
	return out
}

func generateTestFaultInjectors(count int) ([]*spec.FaultInjector, error) {
	names := []string{
		"actinium", "thorium", "protactinium", "uranium", "neptunium",
		"plutonium", "americium", "curium", "berkelium", "californium",
		"einsteinium", "fermium", "mendelevium", "nobelium", "lawrencium",
	}
	types := []spec.FaultInjectorType{"PodKiller"}
	if count > len(names) {
		return make([]*spec.FaultInjector, 0), errors.New(fmt.Sprint("Maximum FaultInjector list length is", len(names), "but you requested", count))
	}
	var tests []*spec.FaultInjector
	for k := 0; k < count; k++ {
		var namespace string
		labels := make(map[string]string)
		if int(math.Mod(float64(k), float64(2))) == 0 {
			labels["period"] = "seven"
		}
		if int(math.Mod(float64(k), float64(3))) == 0 {
			labels["group"] = "actinides"
		}
		if int(math.Mod(float64(k), float64(2))) == 0 {
			namespace = "test-namespace-one"
		} else {
			namespace = "test-namespace-two"
		}
		specType := types[int(math.Mod(float64(k), float64(len(types))))]
		tests = append(tests, &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      names[k],
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: spec.FaultInjectorSpec{
				Type: specType,
			},
		})
	}
	return tests, nil
}

func prepareResourceHandlerTest() (*FaultInjectorController, *fcache.FakeControllerSource) {
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset()
	c := &FaultInjectorController{
		kclient: clientset,
	}

	clientset.Core().Namespaces().Create(&v1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: "test-namespace-one"},
	})
	clientset.Core().Namespaces().Create(&v1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: "test-namespace-two"},
	})

	resourceHandlers := &cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFaultInjector,
		DeleteFunc: c.handleDeleteFaultInjector,
		UpdateFunc: c.handleUpdateFaultInjector,
	}

	source := fcache.NewFakeControllerSource()

	store, controller := cache.NewInformer(
		source,
		&spec.FaultInjector{},
		time.Millisecond*100,
		resourceHandlers)

	c.store = store
	c.controller = controller

	return c, source
}

func checkResourceEqual(deployment *extensionsobj.Deployment, resource *spec.FaultInjector) bool {
	var expected extensionsobj.Deployment
	expected = *deployment
	err := updateDownstreamObject(&expected, resource)
	return (err == nil && reflect.DeepEqual(&expected, deployment) && resource.ObjectMeta.Namespace == deployment.ObjectMeta.Namespace)
}

func validateSingleResourcePresent(t *testing.T, deploymentList []*extensionsobj.Deployment, resource *spec.FaultInjector) {
	for i := range deploymentList {
		if checkResourceEqual(deploymentList[i], resource) {
			return
		}
	}
	t.Errorf("Did not find expected resource:\n%v", resource)
}

func validateSingleResourceAbsent(t *testing.T, deploymentList []*extensionsobj.Deployment, resource *spec.FaultInjector) {
	for i := range deploymentList {
		if checkResourceEqual(deploymentList[i], resource) {
			t.Errorf("Found resource that should have been deleted:\n%v", resource)
		}
	}
}

func validateResourceList(t *testing.T, deploymentList []*extensionsobj.Deployment, resourceList []*spec.FaultInjector) {
	if len(deploymentList) != len(resourceList) {
		t.Errorf("Expected %v deployments, found %v", len(resourceList), len(deploymentList))
	}
One:
	for i := range deploymentList {
		for j := range resourceList {
			if checkResourceEqual(deploymentList[i], resourceList[j]) {
				continue One
			}
		}
		t.Errorf("Found unexpected resource:\n%v", deploymentList[i])
	}
Two:
	for j := range resourceList {
		for i := range deploymentList {
			if checkResourceEqual(deploymentList[i], resourceList[j]) {
				continue Two
			}
		}
		t.Errorf("Expected to find resource, but didn't:\n%v", resourceList[j])
	}
}
