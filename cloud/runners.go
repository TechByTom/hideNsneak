package cloud

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rmikehodges/SneakyVulture/nmap"
	"github.com/rmikehodges/SneakyVulture/sshext"
)

//Proxies//
func CreateSOCKS(Instances []*Instance, startPort int) (string, string) {
	socksConf := make(map[int]string)
	counter := startPort
	for _, instance := range Instances {
		instance.Proxy.SOCKSActive, instance.Proxy.Process = sshext.CreateSingleSOCKS(instance.SSH.PrivateKey, instance.SSH.Username, instance.Cloud.IPv4, counter)
		if instance.Proxy.SOCKSActive {
			instance.Proxy.SOCKSPort = strconv.Itoa(counter)
			socksConf[counter] = instance.Cloud.IPv4
			counter = counter + 1
		}

	}

	proxychains := sshext.PrintProxyChains(socksConf)
	socksd := sshext.PrintSocksd(socksConf)
	return proxychains, socksd
}

//Nmap Helpers//
//TODO: Add an even more evasive option in here that will further limit the IPs scanned on that one address.
//TODO: Add ability for users to define their scan names further
func RunConnectScans(instances []*Instance, output string, additionalOpts string, evasive bool, scope string, ports []string, localDir string) {
	targets := nmap.ParseIPFile(scope)
	ipPorts := nmap.GenerateIPPortList(targets, ports)
	if evasive {
		fmt.Println("Evasive")
		nmapTargeting := nmap.RandomizeIPPortsToHosts(len(instances), ipPorts)
		for i, instance := range instances {
			go nmap.InitiateConnectScan(instance.SSH.Username, instance.Cloud.IPv4, instance.SSH.PrivateKey, nmapTargeting[i],
				instance.Cloud.HomeDir, localDir, strings.Join(ports, "-"), additionalOpts,
				evasive)
		}
	}
	// else {
	// 	fmt.Println("Less-Evasive")
	// 	splitIPsToHosts(Instances, ports, targets)
	// 	// for i := range Instances {
	// 	// 	 go Instances[i].initiateNmap(output, additionalOpts, false)
	// 	// }
	// }
}

// //This doesn't work very well
// func CheckAllNmapProcesses(ipv4 string, username string, privateKey string, nmapCmd string) {
// 	fmt.Println("See! I checked!")
// 	for {
// 		oneActive := false
// 		for i := range Instances {
// 			if Instances[i].NmapActive {
// 				Instances[i].checkNmapProcess()
// 				oneActive = true
// 			}
// 		}

// 		if !oneActive {
// 			fmt.Println("/////////////////////////No Nmap Running////////////////////")
// 		}
// 		time.Sleep(30 * time.Second)
// 	}
// }
