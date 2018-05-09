# Virtual Network Device Plugin
--------------------------------

## Overview
The is a fork of the sample [Solarwind Device Plugin Repository](https://github.com/vikaschoudhary16/sfc-device-plugin).

The goal of this sample is to demonstrate the ability of inserting a Virtual Network Function (VNF) into the network path
for any Kubernetes POD using standard Kubernetes mechanisms.
    
 Adjust the config map parameters for onload configuration:

    $ cat device_plugins/sfc_nic/device_plugin.yml
    ---
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: configmap
    data:
      onload-version: 201606-u1.3
      reg-exp-sfc: (?m)[\r\n]+^.*SFC[6-9].*$
      socket-name: sfcNIC
      resource-name: solarflare/smartNIC
      k8s-api: http://<master-ip>:8080
      node-label-onload-version: device.sfc.onload-version
  And then deploy the daemonsets:

    $ kubectl apply -f device_plugins/sfc_nic/device-plugin.yml -n kube-system


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

## More Details:
    [Virtual Network Device Plugin](https://docs.google.com/document/d/1_weY_f6j4et56mCZGhbXfCiwvyWxFIwUKl4R0fc1F5c/edit#heading=h.d463l2cyl7wb)
