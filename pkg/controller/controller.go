// Package controller implements the main FaultInjectorController type and
// related methods for creating and running the FaultInjector controller.
package controller

import (
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	apierrors "k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/rest"
)

// FaultInjectorController manages TypeInjector resources.
type FaultInjectorController struct {
	// TODO: proper configuration
	kclient *kubernetes.Clientset
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
func (c *FaultInjectorController) Run() error {
	err := c.createTPR()
	fmt.Println("Started FaultInjector controller")
	return err
}

// Create the FaultInjector ThirdPartyResource in kubernetes.
func (c *FaultInjectorController) createTPR() error {
	tpr := &extensionsobj.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: "fault-injector.k8s.puppet.com",
		},
		Versions: []extensionsobj.APIVersion{
			{
				Name: "v1alpha1",
			},
		},
		Description: "A specification for a fault injection task to run against the service",
	}

	tprClient := c.kclient.Extensions().ThirdPartyResources()

	if _, err := tprClient.Create(tpr); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	restClient := c.kclient.CoreClient.GetRESTClient()

	return wait.Poll(3*time.Second, 30*time.Second, func() (bool, error) {
		resp := restClient.Get().AbsPath("apis", "fault-injector.k8s.puppet.com", "v1alpha1", "faultinjectors").Do()
		err := resp.Error()
		if err != nil {
			if statuserr, ok := err.(*apierrors.StatusError); ok {
				if statuserr.Status().Code == http.StatusNotFound {
					return false, nil
				}
			}
			return false, err
		}
		var statusCode int
		resp.StatusCode(&statusCode)
		if statusCode != http.StatusOK {
			return false, fmt.Errorf("invalid status code: %v", statusCode)
		}
		return true, nil
	})
}
