package nmap

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func cidrHosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func parseIPFile(path string) []string {
	var ipList []string
	var cidrList []string
	var endNum int
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for _, ip := range lines {
		if _, _, err := net.ParseCIDR(ip); err == nil {
			cidrList, _ = cidrHosts(ip)
			ipList = append(ipList, cidrList...)
		}
		if net.ParseIP(ip) != nil {
			ipList = append(ipList, ip)
		}
		if strings.Contains(ip, "-") {

			ipRangeList := strings.Split(ip, "-")
			digitList := strings.Split(ipRangeList[0], ".")
			threeNumbers := strings.Join(digitList[:3], ".")
			lastDigit := digitList[3]
			startNum, _ := strconv.Atoi(lastDigit)

			if net.ParseIP(ipRangeList[1]) != nil {
				digitList = strings.Split(ipRangeList[1], ".")
				endNum, _ = strconv.Atoi(digitList[3])
			} else {
				endNum, _ = strconv.Atoi(ipRangeList[1])
			}
			for i := startNum; i <= endNum; i++ {
				incrementToString := strconv.Itoa(i)
				ipList = append(ipList, threeNumbers+"."+incrementToString)
			}
		}
	}
	return ipList
}

func normalizeTargets(targets []string) string {
	return strings.Join(targets, " ")
}

func generateIPPortList(targets []string, ports []string) []string {
	var ipPortList []string
	for _, port := range ports {
		for _, ip := range targets {
			ipPortList = append(ipPortList, ip+":"+port)
		}
	}
	return ipPortList
}

//This is for splitting up hosts more granualarly for stealthier scans
func randomizeIPPortsToHosts(Instances map[int]*Instance, ipPortList []string) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for _, i := range r.Perm(len(ipPortList)) {
		p := i % len(Instances)
		splitArray := strings.Split(ipPortList[i], ":")
		if len(Instances[p].NmapTargets) != 0 {
			Instances[p].NmapTargets[splitArray[1]] = append(Instances[p].NmapTargets[splitArray[1]], splitArray[0])
		} else {
			Instances[p].NmapTargets = make(map[string][]string)
			Instances[p].NmapTargets[splitArray[1]] = strings.Split(splitArray[0], "  ")
		}
	}
}

//This is for splitting up hosts straight up for less stealthy scans
func splitIPsToHosts(Instances map[int]*Instance, portList []string, ipList []string) {
	count := len(Instances)
	splitNum := len(ipList) / count
	for i := range Instances {
		Instances[i].NmapTargets = make(map[string][]string)
		Instances[i].NmapTargets = make(map[string][]string)
		for _, port := range portList {
			if i != count-1 {
				Instances[i].NmapTargets[port] = ipList[i*splitNum : (i+1)*splitNum]
			} else {
				Instances[i].NmapTargets[port] = ipList[i*splitNum:]
			}
		}
	}
}

// func (instance Instance) parseNmapTargets() (portList []string, ipList []string) {
// 	for _, ipPort := range instance.Nmap.NmapTargets{
// 		splitArray := strings.Split(ipPort, ":")
// 		ipList = removeDuplicateStrings(append(ipList, splitArray[0]))
// 		portList = removeDuplicateStrings(append(portList, splitArray[1]))
// 	}
// 	return
// }

func (instance *Instance) initiateConnectScan(outputFile string, additionOpts string, evasive bool) {
	sshConfig := &ssh.ClientConfig{
		User: instance.SSH.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	instance.executeCmd(`sudo apt-get update;
		sudo apt-get install -y nmap;
		sudo apt-get install -y screen`, sshConfig)

	fmt.Println("Successfully installed Nmap")

	instance.System.NmapDir = homedir + "/" + ipv4 + "-nmap"
	fmt.Println("Making directory")
	instance.executeCmd("mkdir "+instance.System.NmapDir, sshConfig)
	if evasive {
		for port, ipList := range instance.Nmap.NmapTargets {
			fmt.Println("In the evasive scanning if statement")
			timestamp := time.Now().Format("20060102150405")
			ips := (strings.Join(ipList, " "))
			instance.Nmap.NmapCmd = "nmap -oA" + " " + instance.System.NmapDir + "/" + timestamp + "_" + outputFile + " " + "-p" + port + " " + additionOpts + " " + ips

			fmt.Println("Executing nmap")
			fmt.Println(instance.Nmap.NmapCmd)

			// PORT SCAN
			command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, instance.SSH.Username+"@"+ipv4,
				"sudo", "nmap", "-oA", instance.System.NmapDir+"/"+timestamp+"_"+outputFile, "-p", port, additionOpts, ips)

			// //PING SCAN
			// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,instance.SSH.Username + "@" + ipv4,
			// 	"sudo", "nmap", "-oA", instance.System.NmapDir + "/" + timestamp + "_" + outputFile, additionOpts, ips  )

			//Cmd Exec run is consuming a lot of memory due to the fact the method must hold.
			if err := command.Run(); err != nil {
				fmt.Println("nmap")
				fmt.Println(err)
			}
		}
	} else {
		var portList []string
		var finalIPList []string
		for port, ipList := range instance.Nmap.NmapTargets {
			portList = append(portList, port)
			finalIPList = append(finalIPList, ipList...)
		}
		portList = removeDuplicateStrings(portList)
		timestamp := time.Now().Format("20060102150405")
		ports := strings.Join(portList, ",")
		ips := strings.Join(finalIPList, " ")
		instance.Nmap.NmapCmd = "nmap -oA " + instance.System.NmapDir + "/" + timestamp + "_" + outputFile + " -p" + ports + " " + additionOpts + " " + ips

		fmt.Println("Executing nmap")
		fmt.Println(instance.Nmap.NmapCmd)

		// PORT SCAN
		command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, instance.SSH.Username+"@"+ipv4,
			"sudo", "nmap", "-oA", instance.System.NmapDir+"/"+timestamp+"_"+outputFile, "-p", ports, additionOpts, ips)

		// //PING SCAN
		// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,instance.SSH.Username + "@" + ipv4,
		// 	"sudo", "nmap", "-oA", instance.System.NmapDir + "/" + timestamp + "_" + outputFile, additionOpts, ips  )
		instance.Nmap.NmapActive = true
		if err := command.Run(); err != nil {
			fmt.Println("nmap")
			fmt.Println(err)
		}
		instance.Nmap.NmapActive = false
	}

	instance.rsyncDirFromHost(instance.System.NmapDir, "/tmp/cloudNmap")
	fmt.Println("done")
}

//This doesn't work very well
func checkAllNmapProcesses(Instances map[int]*Instance) {
	fmt.Println("See! I checked!")
	for {
		oneActive := false
		for i := range Instances {
			if Instances[i].NmapActive {
				Instances[i].checkNmapProcess()
				oneActive = true
			}
		}

		if !oneActive {
			fmt.Println("/////////////////////////No Nmap Running////////////////////")
		}
		time.Sleep(30 * time.Second)
	}
}

func (instance *Instance) checkNmapProcess() {
	sshConfig := &ssh.ClientConfig{
		User: instance.SSH.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	grepString := "ps aux | grep '" + instance.Nmap.NmapCmd + "' | grep -v grep | awk '{print $2}'"

	process := instance.executeCmd(grepString, sshConfig)

	proccessList := strings.Split(process, "\n")

	processString := strings.Join(proccessList[:len(proccessList)-1], ",")
	if len(processString) < 1 {
		instance.Nmap.NmapActive = false
	} else {
		instance.Nmap.NmapActive = true
		instance.Nmap.NmapProcess = processString
	}
}

//TODO: Add an even more evasive option in here that will further limit the IPs scanned on that one address.
func runConnectScans(Instances map[int]*Instance, output string, additionalOpts string, evasive bool, scope string, ports []string) {
	targets := parseIPFile(scope)
	ipPorts := generateIPPortList(targets, ports)
	if evasive {
		fmt.Println("Evasive")
		randomizeIPPortsToHosts(Instances, ipPorts)
		for i := 1; i < len(Instances); i++ {
			go Instances[i].initiateConnectScan(output, additionalOpts, true)
		}
	} else {
		fmt.Println("Less-Evasive")
		splitIPsToHosts(Instances, ports, targets)
		// for i := range Instances {
		// 	 go Instances[i].initiateNmap(output, additionalOpts, false)
		// }
	}

}

//NOTES: Max parallel host size to 1 for more resillient scans based on time window
