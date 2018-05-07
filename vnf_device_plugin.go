package main

import (
	"os"
	//"strconv"
	"bytes"
	"os/exec"
	//"syscall"
	"flag"
	"fmt"
	"github.com/golang/glog"
	//"io/ioutil"
	"net"
	"path"
	//"regexp"
	//"strings"
	"sync"
	"time"
	"strconv"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"
)

//const (
var onloadver string    //  = "0.2"
var onloadsrc string    //= "http://www.openonload.org/download/openonload-" + onloadver + ".tgz"
var socketName string   //= "vnfNIC"
var resourceName string //= "pod.alpha.kubernetes.io/opaque-int-resource-vnfNIC"
var k8sAPI string
var nodeLabelVersion string
var vnfMaxInstances int 	// = 4

//)
// vnfNICManager manages virtual network function devices
type vnfNICManager struct {
	devices     map[string]*pluginapi.Device
	deviceFiles []string
}

func NewVNFNICManager() (*vnfNICManager, error) {
	return &vnfNICManager{
		devices:     make(map[string]*pluginapi.Device),
		deviceFiles: []string{"/dev/zero"},
	}, nil
}

func ExecCommand(cmdName string, arg ...string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("CMD--" + cmdName + ": " + fmt.Sprint(err) + ": " + stderr.String())
	}

	return out, err
}

func (vnf *vnfNICManager) discoverVNFResources(vnfMaxInstances int) bool {
	var vnfName string

	found := true
	for i:=0; i< vnfMaxInstances; i++ {
		//vnf.devices["firewall-"+string(i)] = i
		vnfName = "firewall-" + strconv.Itoa(i)
		dev := pluginapi.Device{ID: vnfName, Health: pluginapi.Healthy}
		vnf.devices[vnfName] = &dev
		found = true
		fmt.Printf("Devices: %v \n", vnf.devices)
	}
	return found
}

func (vnf *vnfNICManager) isOnloadInstallHealthy() bool {
	healthy := true
	
	return healthy
}

func (vnf *vnfNICManager) installOnload() error {
	return nil
}

func (vnf *vnfNICManager) Init() error {
	glog.Info("Init\n")
	err := vnf.installOnload()
	return err
}

func Register(kubeletEndpoint string, pluginEndpoint, socketName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

// Implements DevicePlugin service functions
func (vnf *vnfNICManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Info("device-plugin: ListAndWatch start\n")
	for {
		vnf.discoverVNFResources(vnfMaxInstances)
		if !vnf.isOnloadInstallHealthy() {
			glog.Errorf("Error with onload installation. Marking devices unhealthy.")
		}
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range vnf.devices {
			glog.Info("dev ", dev)
			resp.Devices = append(resp.Devices, dev)
		}
		glog.Info("resp.Devices ", resp.Devices)
		if err := stream.Send(resp); err != nil {
			glog.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (vnf *vnfNICManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var containerID string 
	var containerPID bytes.Buffer
	//
	glog.Info("Allocate rqt: " + rqt + "\n")

	resp := new(pluginapi.AllocateResponse)
	//containerName := strings.Join([]string{"k8s", "POD", rqt.PodName, rqt.Namespace}, "_")
	glog.Info("Container Name: " + containerName + "\n")
	for _, id := range rqt.DevicesIDs {
		if _, ok := vnf.devices[id]; ok {
			for _, d := range vnf.deviceFiles {
				resp.Devices = append(resp.Devices, &pluginapi.DeviceSpec{
					HostPath:      d,
					ContainerPath: d,
					Permissions:   "mrw",
				})
			}
			glog.Info("Allocated interface ", id)
			//glog.Info("Allocate interface ", id, " to ", containerName)
			containerPID, _ = ExecCommand("docker", "-H unix:///gopath/run/docker.sock","inspect","--format","{{ .State.Pid }}", containerID)
			MoveInterface(containerPID.String(),"eth0","crb0")
		}
	}
	return resp, nil
}

func MoveInterface(containerPID string,interfaceName string, bridgeName string) {
	var out bytes.Buffer
	glog.Info("move interface after reading checkpoint file")
	fmt.Printf("Moving Interface of ContainerPID: " + containerPID + "\n")
	out,_ = ExecCommand("nsenter", "-t", containerPID, "-n", "ip", "netns","add", "nsvnf1")
	fmt.Printf("Output of nsenter: "+out.String()+ "\n")
}

func AnnotateNodeWithOnloadVersion(version string) {
	glog.Info("Annotating Node with onload version: ", version, " ", nodeLabelVersion)
}

func AreAllOnloadDevicesAvailable() bool {
	glog.Info("AreAllOnloadDevicesAvailable\n")
	return true
}

func (vnf *vnfNICManager) UnInit() {
	glog.Info("UnInit\n")
	return
}

func main() {
	flag.Parse()
	fmt.Printf("Starting main \n")

	onloadver = os.Args[1] //"201606-u1.3"
	socketName = os.Args[2]   //"sfcNIC"
	resourceName = os.Args[3] //"pod.alpha.kubernetes.io/opaque-int-resource-sfcNIC"
	k8sAPI = os.Args[4]
	nodeLabelVersion = os.Args[5]
	//vnfMaxInstances = os.Args[6]
   	vnfMaxInstances, err := strconv.Atoi(os.Args[6])
    if err != nil {
        // handle error
        glog.Info(err)
        vnfMaxInstances = 4
    }

	flag.Lookup("logtostderr").Value.Set("true")

	vnf, err := NewVNFNICManager()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	found := vnf.discoverVNFResources(vnfMaxInstances)
	if !found {
		// clean up any exisiting device plugin software
		//sfc.UnInit()
		glog.Errorf("No VNFs are present\n")
		os.Exit(1)
	}
	if !vnf.isOnloadInstallHealthy() {
		//err = sfc.Init()
		//if err != nil {
		glog.Errorf("Error with onload installation")
		//		for _, device := range sfc.devices {
		//			device.Health = pluginapi.Unhealthy
		//		}
		//	}
		AnnotateNodeWithOnloadVersion("")
	}
	AnnotateNodeWithOnloadVersion(onloadver)

	pluginEndpoint := fmt.Sprintf("%s-%d.sock", socketName, time.Now().Unix())
	//serverStarted := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	// Starts device plugin service.
	go func() {
		defer wg.Done()
		fmt.Printf("DevicePluginPath %s, pluginEndpoint %s\n", pluginapi.DevicePluginPath, pluginEndpoint)
		fmt.Printf("device-plugin start server at: %s\n", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		lis, err := net.Listen("unix", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		if err != nil {
			glog.Fatal(err)
			return
		}
		grpcServer := grpc.NewServer()
		pluginapi.RegisterDevicePluginServer(grpcServer, vnf)
		grpcServer.Serve(lis)
	}()

	// TODO: fix this
	time.Sleep(5 * time.Second)
	// Registers with Kubelet.
	err = Register(pluginapi.KubeletSocket, pluginEndpoint, resourceName)
	if err != nil {
		glog.Fatal(err)
	}
	fmt.Printf("device-plugin registered\n")
	wg.Wait()
}
