#!/bin/bash
./vnf-device-plugin $ONLOAD_VERSION $SOCKET_NAME $RESOURCE_NAME $K8S_API $NODE_LABEL_ONLOAD_VERSION $VNF_MAX_INSTANCES $K8S_PASSWD -logtostderr=true
