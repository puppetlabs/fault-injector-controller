package podkiller

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
)

// PodKiller deletes Pods from Kubernetes.
type PodKiller struct {
	kclient   kubernetes.Interface
	namespace string
}

// Config holds configuration parameters for a PodKiller.
type Config struct {
	Namespace   string
	Host        string
	TLSInsecure bool
	TLSConfig   rest.TLSClientConfig
}

// New creates a new PodKiller.
func New(conf Config) (*PodKiller, error) {
	var cfg *rest.Config
	var err error

	if len(conf.Host) == 0 {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		cfg = &rest.Config{
			Host: conf.Host,
		}
		hostURL, err := url.Parse(conf.Host)
		if err != nil {
			return nil, fmt.Errorf("Error parsing host URL %s: %v", conf.Host, err)
		}
		if hostURL.Scheme == "https" {
			cfg.TLSClientConfig = conf.TLSConfig
			cfg.Insecure = conf.TLSInsecure
		}
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &PodKiller{
		kclient:   client,
		namespace: conf.Namespace,
	}, nil
}

// Run starts the PodKiller service.
func (p *PodKiller) Run(interval time.Duration, stopChan <-chan struct{}) error {
	wait.Until(p.killPods, interval, stopChan)
	return nil
}

func (p *PodKiller) killPods() {
	allPods, err := p.kclient.Core().Pods(p.namespace).List(api.ListOptions{})
	if err == nil && len(allPods.Items) > 0 {
		podToKill := allPods.Items[rand.Intn(len(allPods.Items))]
		p.kclient.Core().Pods(p.namespace).Delete(podToKill.Name, &api.DeleteOptions{})
	}
}
