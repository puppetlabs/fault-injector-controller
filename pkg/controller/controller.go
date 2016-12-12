// Package controller implements the main FaultInjectorController type and
// related methods for creating and running the FaultInjector controller.
package controller

import (
	"fmt"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	apierrors "k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

var (
	tprGroup   = "k8s.puppet.com"
	tprVersion = version.ResourceAPIVersion
	tprKind    = "faultinjectors"

	tprName = "fault-injector." + tprGroup
)

// FaultInjectorController manages TypeInjector resources.
type FaultInjectorController struct {
	// TODO: proper configuration
	kclient    kubernetes.Interface
	controller *cache.Config
}

// New creates a new controller.
func New() (*FaultInjectorController, error) {
	// TODO: Actually create this.

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &FaultInjectorController{
		kclient: client,
	}, nil
}

// Run starts the FaultInjector controller service.
func (c *FaultInjectorController) Run(stopChan <-chan struct{}) error {
	err := c.createTPR()

	lw := cache.NewListWatchFromClient(
		c.kclient.Core().GetRESTClient(),
		tprKind,
		api.NamespaceAll,
		nil)

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFaultInjector,
		DeleteFunc: c.handleDeleteFaultInjector,
		UpdateFunc: c.handleUpdateFaultInjector,
	}

	store, con := cache.NewInformer(lw, &spec.FaultInjector{}, 0, resourceHandler)
	fmt.Println("Store:", store)
	fmt.Println("Con:", con)
	go con.Run(stopChan)
	fmt.Println("Started FaultInjector controller")
	return err
}

func (c *FaultInjectorController) handleAddFaultInjector(obj interface{}) {
	fmt.Println("Add", obj)
}

func (c *FaultInjectorController) handleDeleteFaultInjector(obj interface{}) {
	fmt.Println("Delete", obj)
}

func (c *FaultInjectorController) handleUpdateFaultInjector(old, cur interface{}) {
	fmt.Println("Update", old, "to", cur)
}

// Create the FaultInjector ThirdPartyResource in kubernetes.
func (c *FaultInjectorController) createTPR() error {
	fmt.Println("Creating ThirdPartyResource")
	tpr := &extensionsobj.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: tprName,
		},
		Versions: []extensionsobj.APIVersion{
			{
				Name: tprVersion,
			},
		},
		Description: "A specification for a fault injection task to run against the service",
	}

	tprClient := c.kclient.Extensions().ThirdPartyResources()

	if _, err := tprClient.Create(tpr); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return wait.Poll(3*time.Second, 30*time.Second, func() (bool, error) {
		fmt.Println("Checking that TPR was created")
		_, err := c.kclient.Extensions().ThirdPartyResources().Get(tprName)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}
