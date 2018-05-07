#!/bin/bash -x
DEVICE_ID=${1}
K8S_API=${2}

sleep 0.1
eval POD_UID=`cat /var/lib/kubelet/device-plugins/kubelet_internal_checkpoint |  jq --arg DEVICE_ID "$DEVICE_ID" '.PodDeviceEntries[] | select(.DeviceIDs[] | contains($DEVICE_ID)) | .PodUID'`

eval POD_NAME=$(curl --insecure --user admin:0gh2qDfU0ImRwfit $K8S_API/api/v1/pods/ | jq -r --arg POD_UID $POD_UID '.items[].metadata | select(.uid == $POD_UID) | .name')
eval POD_NAMESPACE=$(curl --insecure --user admin:0gh2qDfU0ImRwfit  $K8S_API/api/v1/pods/ | jq -r --arg POD_UID $POD_UID '.items[].metadata | select(.uid == $POD_UID) | .namespace')
containerName="k8s_POD_${POD_NAME}_${POD_NAMESPACE}"

#ssh -o StrictHostKeyChecking=no 127.0.0.1 rm -f /var/run/netns/$containerName
#containerID=`docker ps | grep $containerName | awk {'print $1'}`
#containerID=`docker -H unix:///gopath/run/docker.sock ps | grep $containerName | awk {'print $1'}`
eval containerID=$(curl --insecure --user admin:0gh2qDfU0ImRwfit  $K8S_API/api/v1/pods/ | jq -r --arg POD_UID $POD_UID '.items[].metadata | select(.uid == $POD_UID) | .containerID')
echo $containerID
while [ -z $containerID ]; do
        echo "sleep"
        containerID=`docker ps | grep $containerName | awk {'print $1'}`
          sleep 0.1
done
PID=`docker -H unix:///gopath/run/docker.sock inspect --format '{{ .State.Pid }}' $containerID`
echo $PID
