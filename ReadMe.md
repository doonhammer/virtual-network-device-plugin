# Virtual Network Device Plugin
--------------------------------

## Overview
The is a fork of the sample [Solarwind Device Plugin Repository](https://github.com/vikaschoudhary16/sfc-device-plugin).

The goal of this sample is to demonstrate the ability of inserting a Virtual Network Function (VNF) into the network path
for any Kubernetes POD using standard Kubernetes mechanisms. A detailed write up on the goals and approach is available at
[Virtual Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb). This
document is open for comments/


## Deployment

This sample code has been deployed successfully on GKE with Kubernetes v1.9.7. It has not been deployed on any other public or private cloud infrastructure. There should not be any issues on other clouds as the implmentation uses standard (though Alpha) Kubernetes features.

1. Initial Setup of GKE
  1. Assume use has GKE account
  1. Install gcloud

1. Configuring the Kubernetes Cluster

```bash

gcloud alpha container clusters create vnf-demo
  --enable=kubernetes-alpha \
  --cluster-version 1.9.7

``` 

2. Edit the configMap in the device-plugin.yaml file

```yaml

apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap
data:
  onload-version: "0.2"
  socket-name: vnfNIC
  resource-name: paloaltonetworks.com/vnfdevice
  k8s-api: https://<ClusterIP>
  node-label-onload-version: device.vnf.onload-version
  vnf-max-instances: "8"
  k8s-passwd: <cluster credentials password>

```

3. Deploy the device plugin daemonset:

```bash

    $ kubectl apply -f ./device-plugin.yaml

```

4.Sample pod template to consume VNFs

```yaml

apiVersion: v1
kind: Pod
metadata:
  name: nginxtwin
  labels:
    name: webserver
spec:
  containers:
  - name: nginxtwin
    image: nginx
    resources:
      limits:
        paloaltonetworks.com/vnfdevice: '1'

```

```bash

  $ kubectl apply -f ./nginx.yaml

'''

## Current Issues
1. Only possible to get container ID in Allocate method through a workaround.
2. Deallocating resources when a POD is deleted is an issue.
3. Getting an addition interface into the VNF for management. This may be possible using another veth pair and a local IP-Tables rule on a well known port.

## References:

1.[Virtual Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb). 
