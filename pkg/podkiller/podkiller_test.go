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

func TestRun(t *testing.T) {
	objects, err := generatePodList(5)
	if err != nil {
		t.Fatal("Error when generating pods for test:", err)
	}
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset(objects...)

	p := &PodKiller{
		kclient: clientset,
	}

	t.Run("testRunFourSecondInterval", p.testRunGenerator(time.Second*4, 4))
}

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
		defer close(stopChan)

		go p.Run(interval, stopChan)
		for i := 0; i < iterations; i++ {
			if pods, err := p.kclient.Core().Pods(api.NamespaceDefault).List(api.ListOptions{}); err != nil {
				t.Error("Unexpected error when trying to list pods:", err)
			} else if len(pods.Items) != initialPodCount-i {
				t.Error("Expected there to be", initialPodCount-i, "pods after", i+1, "iterations, but found", len(pods.Items))
			}
			// TODO: this fails sometimes when the timing doesn't line up right.
			time.Sleep(interval)
		}
	}
}
