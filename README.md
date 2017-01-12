# Kubernetes FaultInjector

This FaultInjector allows you to deploy a resource into your Kubernetes cluster to intentionally cause failures in your infrastructure in order to test the resilience of your infrastructure, in a manner like Netflix's Simian Army.

## Quick Usage

Set up the controller:

~~~
kubectl apply -f deployment.yaml
~~~

Create a FaultInjector:

~~~
kubectl create -f - << EOF
---
apiVersion: "k8s.puppet.com/v1alpha1"
kind: FaultInjector
metadata:
  name: example-faultinjector
spec:
  type: "PodKiller"
EOF
~~~

Now you should have a PodKiller running in Kubernetes which will kill a random pod every minute.