package podkiller

import (
	"math/rand"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
)

// PodKiller deletes Pods from Kubernetes.
type PodKiller struct {
	kclient kubernetes.Interface
}

// New creates a new PodKiller.
func New() (*PodKiller, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &PodKiller{
		kclient: client,
	}, nil
}

// Run starts the PodKiller service.
func (p *PodKiller) Run(interval time.Duration, stopChan <-chan struct{}) error {
	wait.Until(p.killPods, interval, stopChan)
	return nil
}

func (p *PodKiller) killPods() {
	allPods, err := p.kclient.Core().Pods(api.NamespaceDefault).List(api.ListOptions{})
	if err == nil && len(allPods.Items) > 0 {
		podToKill := allPods.Items[rand.Intn(len(allPods.Items))]
		p.kclient.Core().Pods(api.NamespaceDefault).Delete(podToKill.Name, &api.DeleteOptions{})
	}
}
