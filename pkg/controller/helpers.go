package controller

import (
	"fmt"

	"github.com/puppetlabs/fault-injector-controller/pkg/spec"
	"github.com/puppetlabs/fault-injector-controller/version"

	"k8s.io/client-go/1.5/pkg/api/v1"
	extensionsobj "k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

func generateDownstreamObject(obj *spec.FaultInjector) (*extensionsobj.Deployment, error) {
	containers, err := generateDownstreamContainers(obj)
	if err != nil {
		return nil, err
	}
	labels := generateDownstreamLabels(obj)

	deploymentObj := &extensionsobj.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      formatDownstreamName(obj),
			Namespace: obj.ObjectMeta.Namespace,
			Labels:    map[string]string{"generatedBy": "FaultInjector"},
		},
		Spec: extensionsobj.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: containers,
					Volumes: []v1.Volume{
						v1.Volume{
							Name: "podinfo",
							VolumeSource: v1.VolumeSource{
								DownwardAPI: &v1.DownwardAPIVolumeSource{
									Items: []v1.DownwardAPIVolumeFile{
										v1.DownwardAPIVolumeFile{
											Path: "namespace",
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.namespace",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return deploymentObj, nil
}

func updateDownstreamObject(downstreamObj *extensionsobj.Deployment, newObj *spec.FaultInjector) error {
	if downstreamObj.ObjectMeta.Name != formatDownstreamName(newObj) {
		return fmt.Errorf("Expected downstream object to have the same name as upstream object (%v), but got %v",
			formatDownstreamName(newObj),
			downstreamObj.ObjectMeta.Name)
	}
	containers, err := generateDownstreamContainers(newObj)
	if err != nil {
		return err
	}
	labels := generateDownstreamLabels(newObj)
	downstreamObj.Spec.Template.ObjectMeta.Labels = labels
	downstreamObj.Spec.Template.Spec.Containers = containers
	return nil
}

func formatDownstreamName(obj *spec.FaultInjector) string {
	return fmt.Sprintf("faultinjector-%v", obj.ObjectMeta.Name)
}

func generateDownstreamContainers(obj *spec.FaultInjector) ([]v1.Container, error) {
	var containers []v1.Container
	switch obj.Spec.Type {
	case "PodKiller":
		containers = append(containers, v1.Container{
			Name:  "fault-injector-podkiller",
			Image: fmt.Sprintf("%v/fault-injector-podkiller:%v", imagePrefix, version.Version),
			Args:  []string{"-namespace-file", "/etc/namespace"},
			VolumeMounts: []v1.VolumeMount{
				v1.VolumeMount{
					Name:      "podinfo",
					MountPath: "/etc",
					ReadOnly:  false,
				},
			},
		})
	default:
		return nil, fmt.Errorf("Unsupported value %v for spec.type on the FaultInjector", obj.Spec.Type)
	}
	return containers, nil
}

func generateDownstreamLabels(obj *spec.FaultInjector) map[string]string {
	labels := make(map[string]string)
	for k, v := range obj.ObjectMeta.Labels {
		labels[k] = v
	}
	labels["faultinjector-type"] = string(obj.Spec.Type)
	return labels
}
