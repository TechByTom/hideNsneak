package cloud

import (
	"fmt"
	"strconv"

	"github.com/rmikehodges/hideNsneak/drone"
	"github.com/rmikehodges/hideNsneak/misc"
	"github.com/rmikehodges/hideNsneak/nmap"
	"github.com/rmikehodges/hideNsneak/sshext"
)

//Proxies//
func CreateSOCKS(Instances []*Instance, startPort int) (string, string) {
	socksConf := make(map[int]string)
	counter := startPort
	for _, instance := range Instances {
		instance.Proxy.SOCKSActive, instance.Proxy.Process = sshext.CreateSingleSOCKS(instance.SSH.PrivateKey, instance.SSH.Username, instance.Cloud.IPv4, counter)
		if instance.Proxy.SOCKSActive {
			misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " SOCKS Created")
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
func RunConnectScans(instances []*Instance, output string, additionalOpts string, evasive bool, scope string,
	ports []string, localDir string, droneImport bool) {
	targets := nmap.ParseIPFile(scope)
	ipPorts := nmap.GenerateIPPortList(targets, ports)
	if evasive == true {
		fmt.Println("Evasive")
		nmapTargeting := nmap.RandomizeIPPortsToHosts(len(instances), ipPorts)
		for i, instance := range instances {
			go nmap.InitiateConnectScan(instance.SSH.Username, instance.Cloud.IPv4, instance.SSH.PrivateKey, nmapTargeting[i],
				instance.Cloud.HomeDir, localDir, additionalOpts, evasive, inhstance.Cloud.Type, innstance.Cloud.Region)

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

func ImportNmaps(localDir string, insecureSSL bool, limitHosts bool, forcePorts bool, lairPID string, tags string) {
	importResult := false
	xmlFiles := nmap.ListNmapXML(localDir)
	for _, xmlFile := range xmlFiles {
		for !importResult {
			importResult = drone.NmapImport(insecureSSL, limitHosts, forcePorts, xmlFile, lairPID, tags)
		}
	}
}

func CreateFirewall(instances []*Instance, config Config, ports []int, groupName string, desc string)  {
	for _, instance := range instances {
		switch instance.Cloud.Type {
		case: "AWS"
			
		case: "DO"
		case: "Google"
		case: "Azure"
		default:
			fmt.Println("Unknown instance type, skpping")
		}
	}
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
