// Package controller implements the main FaultInjectorController type and
// related methods for creating and running the FaultInjector controller.
package controller

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	apierrors "k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/pkg/selection"
	"k8s.io/client-go/1.5/pkg/util/sets"
	"k8s.io/client-go/1.5/pkg/util/wait"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

var (
	tprGroup   = "k8s.puppet.com"
	tprVersion = version.ResourceAPIVersion
	tprKind    = "faultinjectors"

	tprName = "fault-injector." + tprGroup

	imagePrefix = version.ImageRepo
)

// FaultInjectorController manages TypeInjector resources.
type FaultInjectorController struct {
	// TODO: proper configuration
	kclient    kubernetes.Interface
	ficlient   *rest.RESTClient
	store      cache.Store
	controller cache.ControllerInterface
}

// Config holds configuration parameters for a FaultInjectorController.
type Config struct {
	Host        string
	TLSInsecure bool
	TLSConfig   rest.TLSClientConfig
}

type jsonFaultInjectorDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *jsonFaultInjectorDecoder) Close() {
	d.close()
}

func (d *jsonFaultInjectorDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object spec.FaultInjector
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}

// New creates a new controller.
func New(conf Config) (*FaultInjectorController, error) {
	var cfg *rest.Config
	var err error

	c := &FaultInjectorController{}

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
	c.kclient = client

	cfg.APIPath = "/apis"
	cfg.GroupVersion = &unversioned.GroupVersion{
		Group:   tprGroup,
		Version: tprVersion,
	}
	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	ficlient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}
	c.ficlient = ficlient

	lw := prepareListWatch(client, ficlient)
	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddFaultInjector,
		DeleteFunc: c.handleDeleteFaultInjector,
		UpdateFunc: c.handleUpdateFaultInjector,
	}
	store, controller := cache.NewInformer(lw, &spec.FaultInjector{}, 0, resourceHandler)
	c.store = store
	c.controller = controller

	return c, nil
}

// Run starts the FaultInjector controller service.
func (c *FaultInjectorController) Run(stopChan <-chan struct{}) error {
	err := c.createTPR()
	if err != nil {
		return err
	}

	err = c.prepareInitialStore()
	if err != nil {
		return err
	}
	c.controller.Run(stopChan)
	return nil
}

func prepareListWatch(kclient *kubernetes.Clientset, ficlient *rest.RESTClient) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options api.ListOptions) (runtime.Object, error) {
			req := ficlient.Get().
				Namespace(api.NamespaceAll).
				Resource(tprKind)
			b, err := req.DoRaw()
			if err != nil {
				return nil, err
			}
			var l spec.FaultInjectorList
			return &l, json.Unmarshal(b, &l)
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			req := ficlient.Get().
				Prefix("watch").
				Namespace(api.NamespaceAll).
				Resource(tprKind)
			stream, err := req.Stream()
			if err != nil {
				return nil, err
			}
			return watch.NewStreamWatcher(&jsonFaultInjectorDecoder{
				dec:   json.NewDecoder(stream),
				close: stream.Close,
			}), nil
		},
	}
}

func (c *FaultInjectorController) prepareInitialStore() error {
	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement("generatedBy", selection.Equals, sets.NewString("FaultInjector"))
	if err != nil {
		return err
	}
	selector = selector.Add(*requirement)
	listOptions := api.ListOptions{
		LabelSelector: selector,
	}
	deployments, err := c.kclient.Extensions().Deployments(v1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	for i := range deployments.Items {
		name := strings.SplitN(deployments.Items[i].ObjectMeta.Name, "-", 2)
		if name[0] != "faultinjector" {
			return fmt.Errorf("Found an existing deployment that does not match expected name format: %v", deployments.Items[i].ObjectMeta.Name)
		}
		specType := spec.FaultInjectorType(deployments.Items[i].Spec.Template.ObjectMeta.Labels["faultinjector-type"])
		resourceLabels := deployments.Items[i].Spec.Template.ObjectMeta.Labels
		delete(resourceLabels, "faultinjector-type")
		resource := &spec.FaultInjector{
			ObjectMeta: v1.ObjectMeta{
				Name:      name[1],
				Namespace: deployments.Items[i].ObjectMeta.Namespace,
				Labels:    resourceLabels,
			},
			Spec: spec.FaultInjectorSpec{
				Type: specType,
			},
		}
		c.store.Add(resource)
	}
	return nil
}

func (c *FaultInjectorController) handleAddFaultInjector(obj interface{}) {
	var newObj *spec.FaultInjector
	newObj = obj.(*spec.FaultInjector)
	err := c.addFaultInjector(newObj)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (c *FaultInjectorController) handleDeleteFaultInjector(obj interface{}) {
	var newObj *spec.FaultInjector
	switch obj.(type) {
	case *spec.FaultInjector:
		newObj = obj.(*spec.FaultInjector)
	case cache.DeletedFinalStateUnknown:
		newObj = obj.(cache.DeletedFinalStateUnknown).Obj.(*spec.FaultInjector)
	}

	err := c.deleteFaultInjector(newObj)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (c *FaultInjectorController) handleUpdateFaultInjector(old, cur interface{}) {
	var newObj *spec.FaultInjector
	newObj = cur.(*spec.FaultInjector)
	err := c.addFaultInjector(newObj)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (c *FaultInjectorController) addFaultInjector(newObj *spec.FaultInjector) error {
	var err error
	downstreamObj := c.getDownstreamState(newObj)

	if downstreamObj == nil {
		downstreamObj, err = generateDownstreamObject(newObj)
		if err != nil {
			return err
		}
		downstreamObj, err = c.kclient.Extensions().Deployments(downstreamObj.ObjectMeta.Namespace).Create(downstreamObj)
		if err != nil {
			return err
		}
	} else {
		err = updateDownstreamObject(downstreamObj, newObj)
		if err != nil {
			return err
		}
		downstreamObj, err = c.kclient.Extensions().Deployments(downstreamObj.ObjectMeta.Namespace).Update(downstreamObj)
		if err != nil {
			return err
		}
	}
	downstreamObj = c.getDownstreamState(newObj)
	return err
}

func (c *FaultInjectorController) deleteFaultInjector(obj *spec.FaultInjector) error {
	downstreamObj := c.getDownstreamState(obj)
	if downstreamObj != nil {
		err := c.kclient.Extensions().Deployments(downstreamObj.ObjectMeta.Namespace).Delete(downstreamObj.ObjectMeta.Name, &api.DeleteOptions{})
		return err
	}
	return nil
}

func (c *FaultInjectorController) getDownstreamState(obj *spec.FaultInjector) *extensionsobj.Deployment {
	deployment, err := c.kclient.Extensions().Deployments(obj.ObjectMeta.Namespace).Get(formatDownstreamName(obj))
	if err != nil {
		return nil
	}
	return deployment
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

	err := wait.Poll(3*time.Second, 30*time.Second, func() (bool, error) {
		fmt.Println("Checking that TPR was created")
		_, err := c.kclient.Extensions().ThirdPartyResources().Get(tprName)
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return err
}
