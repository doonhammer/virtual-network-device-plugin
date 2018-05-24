# Cloud Network Device Plugin
--------------------------------

## Overview
The is a fork of the sample [Solarwind Device Plugin Repository](https://github.com/vikaschoudhary16/sfc-device-plugin).

The goal of this sample is to demonstrate the ability of inserting a Virtual Network Function (VNF) into the network path
for any Kubernetes POD using standard Kubernetes mechanisms. A detailed write up on the goals and approach is available at
[Virtual Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb). This
document is open for comments/


## Deployment

This sample code has been deployed successfully on GKE with Kubernetes v1.9.6 (Note there appears to be some issues on 1.9.7). It has not been deployed on any other public or private cloud infrastructure. There should not be any issues on other clouds as the implmentation uses standard (though Alpha) Kubernetes features.

1.Initial Setup of GKE
  1.Assume user has GKE account.
  2.Assume gcloud is installed.

2.Configuring the Kubernetes Cluster

```bash

$ gcloud alpha container clusters create vnf-demo
  --enable=kubernetes-alpha \
  --cluster-version 1.9.6

``` 

3.Edit the configMap in the device-plugin.yaml file

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

4.Get a sample CNF

  see [Cloud Network Function](https://github.com/doonhammer/Cloud-Network-Function)

3.Build the device plugin

```bash
$ dep init
$ dep ensure
$ go build -o vnf-device-plugin
$ cp <location of CNF> ./vnf
$ sudo docker build -t gcr.io/<your account>/vnfdevice:0.0.1 .
$ gcloud docker -- push gcr.io/<your account>/vnfdevice:0.0.1
```
4.Deploy the device plugin daemonset:

```bash

    $ kubectl apply -f ./device-plugin.yaml

```

5.Sample pod template to consume VNFs

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

6.Deploy the sample POD 
```bash

  $ kubectl apply -f ./nginx.yaml
```

7.Get the URL
```bash
$ curl http://<nginx pod>
```

8.Check the logs for the VNF
```bash
$ kubectl exec -it <daemonset POD> -- /bin/bash
$ cat /var/log/vnf/vnf.log
```
In the logs you will see the TCP packets from the hot running the curl command.

## Current Issues
1. Only possible to get container ID in Allocate method through a workaround.
2. Deallocating resources when a POD is deleted is an issue.
3. Getting an addition interface into the VNF for management. This may be possible using another veth pair and a local IP-Tables rule on a well known port.

## References:

1.[Cloud Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb). 
