package podkiller

import (
	"errors"
	"fmt"
	"testing"
	"time"

	fkubernetes "k8s.io/client-go/1.5/kubernetes/fake"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/runtime"
)

// TestRun tests the PodKiller.Run() method to validate that it kills pods periodically as expected.
func TestRun(t *testing.T) {
	objects, err := generatePodList(10)
	if err != nil {
		t.Fatal("Error when generating pods for test:", err)
	}
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset(objects...)

	p := &PodKiller{
		kclient: clientset,
	}

	t.Run("testRunFourSecondInterval", p.testRunGenerator(time.Second*4, 4))
	t.Run("testRunSixSecondInterval", p.testRunGenerator(time.Second*6, 2))
}

// TestKillPods tests the PodKiller.killPods() method to validate that it kills exactly one pod each time it is called.
func TestKillPods(t *testing.T) {
	objects, err := generatePodList(3)
	if err != nil {
		t.Fatal("Error when generating pods for test:", err)
	}
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset(objects...)

	p := &PodKiller{
		kclient: clientset,
	}

	p.killPods()
	if pods, err := clientset.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
		t.Error("Found unexpected error when trying to list pods:", err)
	} else if len(pods.Items) != 2 {
		t.Error("Expected that calling killPods() once would leave 2 pods remaining, but found", len(pods.Items))
	}

	p.killPods()
	if pods, err := clientset.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
		t.Error("Found unexpected error when trying to list pods:", err)
	} else if len(pods.Items) != 1 {
		t.Error("Expected that calling killPods() twice would leave 1 pods remaining, but found", len(pods.Items))
	}

	p.killPods()
	if pods, err := clientset.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
		t.Error("Found unexpected error when trying to list pods:", err)
	} else if len(pods.Items) != 0 {
		t.Error("Expected that calling killPods() thrice would leave 0 pods remaining, but found", len(pods.Items))
	}

	p.killPods()
	if pods, err := clientset.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
		t.Error("Found unexpected error when trying to list pods:", err)
	} else if len(pods.Items) != 0 {
		t.Error("Expected that calling killPods() a fourth time would leave 0 pods remaining, but found", len(pods.Items))
	}
}

// generatePodList generates an array of v1.Pod objects which can be used to instantiate a SimpleClientSet for testing.
func generatePodList(count int) ([]runtime.Object, error) {
	names := []string{
		"lanthanum", "cerium", "praseodymium", "neodymium", "promethium",
		"samarium", "europium", "gadolinium", "terbium", "dysprosium",
		"holmium", "erbium", "thulium", "ytterbium", "lutetium",
	}
	if count > len(names) {
		return make([]runtime.Object, 0), errors.New(fmt.Sprint("Maximum pod list length is", len(names), "but you requested", count))
	}
	podList := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		podList[i] = &v1.Pod{ObjectMeta: v1.ObjectMeta{Name: names[i], Namespace: v1.NamespaceDefault}}
	}
	return podList, nil
}

// testRunGenerator returns a function that will call p.Run() and then check that the expected number of pods were killed.
func (p *PodKiller) testRunGenerator(interval time.Duration, iterations int) func(*testing.T) {
	return func(t *testing.T) {
		initialPods, err := p.kclient.Core().Pods(api.NamespaceDefault).List(api.ListOptions{})
		if err != nil {
			t.Fatal("Found unexpected error when trying to list pods:", err)
		}
		initialPodCount := len(initialPods.Items)
		if initialPodCount < iterations {
			t.Fatal("Cannot test", iterations, "iterations with only", initialPodCount, "pods.")
		}
		stopChan := make(chan struct{})

		go p.Run(interval, stopChan)
		time.Sleep(interval * time.Duration(iterations))
		close(stopChan)
		if pods, err := p.kclient.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
			t.Error("Unexpected error when trying to list pods:", err)
		} else if len(pods.Items) != initialPodCount-iterations {
			t.Error("Expected there to be", initialPodCount-iterations, "pods after", iterations, "iterations, but found", len(pods.Items))
		}
	}
}
