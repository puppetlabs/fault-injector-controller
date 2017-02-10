package podkiller

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"math"

	"k8s.io/client-go/1.5/kubernetes"
	fkubernetes "k8s.io/client-go/1.5/kubernetes/fake"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/runtime"
)

// TestRun tests the PodKiller.Run() method to validate that it kills pods periodically as expected.
func TestRun(t *testing.T) {
	for _, k := range []struct {
		seconds    int
		iterations int
	}{{4, 4}, {6, 2}} {
		interval := time.Duration(k.seconds) * time.Second
		iterations := k.iterations
		t.Run(fmt.Sprintf("Interval-%v-Seconds", k.seconds), func(t *testing.T) {
			podCount := iterations + 2
			objects, err := generatePodList(podCount)
			if err != nil {
				t.Fatal("Error when generating pods for test:", err)
			}
			var clientset *fkubernetes.Clientset
			clientset = fkubernetes.NewSimpleClientset(objects...)

			p := &PodKiller{
				kclient:   clientset,
				namespace: "pod-namespace",
			}
			stopChan := make(chan struct{})

			validatePodCount(t, p.kclient, podCount, 0)
			go p.Run(interval, stopChan)
			time.Sleep(interval * time.Duration(iterations))
			close(stopChan)
			validatePodCount(t, p.kclient, podCount, iterations)
		})
	}
}

// TestKillPods tests the PodKiller.killPods() method to validate that it kills exactly one pod each time it is called.
func TestKillPods(t *testing.T) {
	podCount := 3
	objects, err := generatePodList(podCount)
	if err != nil {
		t.Fatal("Error when generating pods for test:", err)
	}
	var clientset *fkubernetes.Clientset
	clientset = fkubernetes.NewSimpleClientset(objects...)

	p := &PodKiller{
		kclient:   clientset,
		namespace: "pod-namespace",
	}

	for i := 0; i < podCount+2; i++ {
		t.Run(fmt.Sprintf("KillIteration-%v", i), func(t *testing.T) {
			validatePodCount(t, clientset, podCount, i)
		})
		p.killPods()
	}
}

func validatePodCount(t *testing.T, clientset kubernetes.Interface, initialPodCount int, killInvocations int) {
	expectedCount := int(math.Max(float64(initialPodCount-killInvocations), 0.0))
	if namespacePods, err := clientset.Core().Pods("pod-namespace").List(api.ListOptions{}); err != nil {
		t.Errorf("Found unexpected error when trying to list pods: %v\n", err)
	} else if len(namespacePods.Items) != expectedCount {
		t.Errorf("Expected that calling killPods() %v times would leave %v pods remaining, but found %v\n", killInvocations, expectedCount, len(namespacePods.Items))
	}
	if otherPods, err := clientset.Core().Pods("other-namespace").List(api.ListOptions{}); err != nil {
		t.Errorf("Found unexpected error when trying to list pods: %v\n", err)
	} else if len(otherPods.Items) != initialPodCount {
		t.Errorf("Expected that no other-namespace pods would be killed, but found %v pods were killed\n", initialPodCount-len(otherPods.Items))
	}
}

// generatePodList generates an array of v1.Pod objects which can be used to instantiate a SimpleClientSet for testing.
// It generates 2 * "count" pods: "count" pods in "pod-namespace" and "count" pods in "other-namespace".
func generatePodList(count int) ([]runtime.Object, error) {
	podNames := []string{
		"lanthanum", "cerium", "praseodymium", "neodymium", "promethium",
		"samarium", "europium", "gadolinium", "terbium", "dysprosium",
		"holmium", "erbium", "thulium", "ytterbium", "lutetium",
	}
	otherNames := []string{
		"actinium", "thorium", "protactinium", "uranium", "neptunium",
		"plutonium", "americium", "curium", "berkelium", "californium",
		"einsteinium", "fermium", "mendelevium", "nobelium", "lawrencium",
	}
	maxPods := int(math.Min(float64(len(podNames)), float64(len(otherNames))))
	if count > maxPods {
		return make([]runtime.Object, 0), errors.New(fmt.Sprint("Maximum pod list length is", maxPods, "but you requested", count))
	}
	podList := []runtime.Object{
		&v1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: "pod-namespace"},
		},
		&v1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: "other-namespace"},
		},
	}
	for i := 0; i < count; i++ {
		podList = append(podList, &v1.Pod{ObjectMeta: v1.ObjectMeta{Name: podNames[i], Namespace: "pod-namespace"}})
		podList = append(podList, &v1.Pod{ObjectMeta: v1.ObjectMeta{Name: otherNames[i], Namespace: "other-namespace"}})
	}
	return podList, nil
}
