package main

import (
	"os"
	"strconv"
	"bytes"
	"os/exec"
	"syscall"
	"strings"
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
	"unsafe"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"
        "github.com/vishvananda/netlink"
        "github.com/vishvananda/netns"
)

//const (
var onloadver string    //  = "0.2"
var onloadsrc string    //= "http://www.openonload.org/download/openonload-" + onloadver + ".tgz"
var socketName string   //= "vnfNIC"
var resourceName string //= "pod.alpha.kubernetes.io/opaque-int-resource-vnfNIC"
var k8sAPI string
var nodeLabelVersion string
var vnfMaxInstances int 	// = 8
var k8SPasswd string

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
		vnfName = "vnf-" + strconv.Itoa(i)
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
func (vnf *vnfNICManager) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Info("device-plugin: ListAndWatch start\n")
	for {
		vnf.discoverVNFResources(vnfMaxInstances)
		if !vnf.isOnloadInstallHealthy() {
			glog.Errorf("Error with onload installation. Marking devices unhealthy.")
		}
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range vnf.devices {
			//glog.Info("dev ", dev)
			resp.Devices = append(resp.Devices, dev)
		}
		//glog.Info("resp.Devices ", resp.Devices)
		if err := stream.Send(resp); err != nil {
			glog.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (vnf *vnfNICManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	//var containerID string 
	//var containerPID bytes.Buffer
	//
	glog.Info("Allocate\n")

	resp := new(pluginapi.AllocateResponse)
	//containerName := strings.Join([]string{"k8s", "POD", rqt.PodName, rqt.Namespace}, "_")
	//glog.Info("Container Name: " + containerName + "\n")
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
			go MoveInterface(id)
		}
	}
	return resp, nil
}

func interfaceExists(name string) bool {
	_, err := net.InterfaceByName(name)

	return err == nil
}

func getbridgeIf(name string) (*net.Interface, error) {
	if interfaceExists(name) {
		glog.Info("Found the Bridge device")
		return net.InterfaceByName(name)
        }
	return net.InterfaceByName(name)
}

func vethInterfacesByName(hostVethName, containerVethName string) (*net.Interface, *net.Interface, error) {
	hostVeth, err := net.InterfaceByName(hostVethName)
	if err != nil {
		return nil, nil, err
	}

	containerVeth, err := net.InterfaceByName(containerVethName)
	if err != nil {
		return nil, nil, err
	}

	return hostVeth, containerVeth, nil
}

func createvethPair(namePrefix string) (*net.Interface, *net.Interface, error) {

	hostVethName := fmt.Sprintf("%s0", namePrefix)
	containerVethName := fmt.Sprintf("%s1", namePrefix)

	if interfaceExists(hostVethName) {
		return vethInterfacesByName(hostVethName, containerVethName)
	}

	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = hostVethName

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  containerVethName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return nil, nil, err
	}

	if err := netlink.LinkSetUp(veth); err != nil {
		return nil, nil, err
	}

	ethtoolTXOff(hostVethName)
	ethtoolTXOff(containerVethName)
	return vethInterfacesByName(hostVethName, containerVethName)
}

func attachhostIfBr(bridge, hostVeth *net.Interface) error {
	mtu := 1460
	bridgeLink, err := netlink.LinkByName(bridge.Name)
	if err != nil {
		return err
	}

	hostVethLink, err := netlink.LinkByName(hostVeth.Name)
	if err != nil {
		return err
	}

	/*
	 * Set the MTU of the host link to 1460 for GCE
	 */
	netlink.LinkSetMTU(hostVethLink, mtu)

	return netlink.LinkSetMaster(hostVethLink, bridgeLink.(*netlink.Bridge))
}

func attachcntrIfnewns(cntrVeth *net.Interface, newns string) error {
	mtu := 1460
	fd,_ := netns.GetFromName(newns)

	cntrVethLink,_  := netlink.LinkByName(cntrVeth.Name)

	/*
	 * Set the MTU of the host link to 1460 for GCE
	 */
	netlink.LinkSetMTU(cntrVethLink, mtu)

	return netlink.LinkSetNsFd(cntrVethLink, int(fd))	
}


func movevethpairnewns(containerPID int, newns string) {
	fmt.Printf("In movevethpairnewns().....\n")
	/*
	 * Get the original host namespace
	 */
	origns, _ := netns.Get()	
	/*
	 * Get the handle to network ns of Container pid
	 */
	fd,_ := netns.GetFromPid(containerPID)

	/*
	 * Set the namespace to this handle
	 */
	netns.Set(fd)

	/*
	 * Get the interfaces in that namespace
	 */
	faces, _ := net.Interfaces()

	/*
	 * Get the index of the interface
	 */
	hostVethLink,_  := netlink.LinkByName(faces[1].Name)
	ethtoolTXOff(faces[1].Name)
	parentIndex := hostVethLink.Attrs().ParentIndex

	/*
	 * Set back to original namespace, Get the veth pair
	 */
	netns.Set(origns)
	faceshost, _ := net.Interfaces()

	/*
	 * Move the veth peer to new namespace
	 */
	var vethpeername string
	for _,face := range(faceshost) {
		if face.Index == parentIndex {
			vethpeername = face.Name
			attachcntrIfnewns(&face,newns)
		}	
	}

	/*
	 * Now both the interfaces have been moved.
	 * Change the names of the interfaces in newns
	 * and bring the link up
	 */
	fdnew,_ := netns.GetFromName(newns)
	netns.Set(fdnew)
	facesnew, _ := net.Interfaces()
	fmt.Println(facesnew)
	for _,facenew := range(facesnew) {
		ifname := facenew.Name
		/*
		 * Get the Link to the interface
		 */
		if ifname != "lo" {
			iflink,_ := netlink.LinkByName(ifname)
			/*
			 * The original veth peer would be eth2 for firewall
			 * and the new veth pair interface would be eth1.
			 * Rename the interface before bringing the link UP
			 */
			if facenew.Name == vethpeername {
				netlink.LinkSetName(iflink, "eth2")
			} else {
				netlink.LinkSetName(iflink, "eth1")
			}
			netlink.LinkSetUp(iflink)
		}
	}	
	 
	/*
	 * Change back to original net namespace
	 */
	netns.Set(origns)
}

const (
	SIOCETHTOOL     = 0x8946     // linux/sockios.h
	ETHTOOL_GTXCSUM = 0x00000016 // linux/ethtool.h
	ETHTOOL_STXCSUM = 0x00000017 // linux/ethtool.h
	IFNAMSIZ        = 16         // linux/if.h
)

/*
 * linux/if.h 'struct ifreq'
 */
type ifReqData struct {
	name [IFNAMSIZ]byte
	data uintptr
}

/*
 * Taken from linux/ethtool.h 'struct ethtool_value'
 */
type ethtoolValue struct {
	cmd  uint32
	data uint32
}

func ioctlEthtool(fd int, argp uintptr) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(SIOCETHTOOL), argp)
	if errno != 0 {
		return errno
	}
	return nil
}

/*
 * Disable TX checksum offload on specified interface
 */
func ethtoolTXOff(name string) error {
	if len(name)+1 > IFNAMSIZ {
		return fmt.Errorf("name too long")
	}

	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(socket)

	// Request current value
	value := ethtoolValue{cmd: ETHTOOL_GTXCSUM}
	request := ifReqData{data: uintptr(unsafe.Pointer(&value))}
	copy(request.name[:], name)

	if err := ioctlEthtool(socket, uintptr(unsafe.Pointer(&request))); err != nil {
		return err
	}
	if value.data == 0 { // if already off, don't try to change
		return nil
	}

	value = ethtoolValue{ETHTOOL_STXCSUM, 0}
	return ioctlEthtool(socket, uintptr(unsafe.Pointer(&request)))
}

func MoveInterface(id string) {
	//var cpid bytes.Buffer
	var containerPID string
	newns := "ns" + id

	glog.Info("move interface after reading checkpoint file: ", id, " K8sAPI: ", k8sAPI )
	cpid, err := ExecCommand("/usr/bin/get_container_pid.sh", id, k8sAPI, k8SPasswd)
	if err != nil {
		glog.Error(err)
	}
	containerPID = strings.TrimSuffix(cpid.String(), "\n")
	glog.Info("Container PID: " , containerPID , "\n")

	/*
	 * Create a new network namespace
	 */
	out, err := ExecCommand("nsenter", "-t", containerPID, "-n", "ip", "netns","add", newns)
	if err != nil {
		glog.Error(err)
	}
	out, err = ExecCommand("nsenter", "-t", containerPID, "-n","ip", "addr", "show")
	if err != nil {
		glog.Error(err)
	}
	glog.Info("Output of ip addr show: ", out.String(), "\n")

	/*
	 * Check if the bridge interface exists or not
	 */
	bridgeIf,_ := getbridgeIf("cbr0")

	/*
	 * Create a veth pair
	 */
	vethhostIf, vethcntrIf, _ := createvethPair(id)

	/*
	 * Attach one end to the bridge and the other end to container NS.
	 * Currently there is no method to attach to a newly created NS.
	 * So am using the execCommand for now.
	 */
	attachhostIfBr(bridgeIf, vethhostIf)
	attachcntrIfnewns(vethcntrIf, newns)

	fmt.Println(bridgeIf, vethhostIf, vethcntrIf)

	/*
	 * We also need to get the index of the interface in pod NS,
	 * Gets its pair and move the pair to the new NS
	 */
	pid,_ := strconv.Atoi(containerPID)
	movevethpairnewns(pid, newns)
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
    k8SPasswd = os.Args[7]
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
