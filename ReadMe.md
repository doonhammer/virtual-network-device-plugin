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

  $  gcloud alpha container clusters create vnf-demo
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
  k8s-api: https://10.11.240.1
  node-label-onload-version: device.vnf.onload-version
  vnf-max-instances: "8"
  k8s-passwd: iyJ3gmowug63Zm0q
```

3. Deloy the device plugin daemonset:

    $ kubectl apply -f device-plugin.yml -n kube-system

4.

### Verify if NICs got picked up by plugin and reported fine to kubelet

    [root@dell-r620-01 kubernetes]# kubectl get nodes -o json | jq     '.items[0].status.capacity'
    {
    "cpu": "16",
    "memory": "131816568Ki",
    "solarflare/smartNIC": "2",
    "pods": "110"
    }

## sample pod template to consume VNFs
    apiVersion: v1
    kind: Pod
    metadata:
      name: my.pod1
      annotations:
        sfc-nic-ip: 70.70.70.1/24
    spec:
        containers:
        - name: demo1
        image: sfc-dev-plugin:latest
        imagePullPolicy: Never
        resources:
            requests:
                paloaltonetworks.com/vnf: '1'
            limits:
                paloaltonetworks.com/vnf: '1'

## Current Issues
1. Only possible to get container ID in Allocate method through a workaround.
2. Deallocating resources when a POD is deleted is an issue.

## References:

1.[Virtual Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb). 
