package nmap

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func (instance *Instance) initiateConnectScan(outputFile string, additionOpts string, evasive bool) {
	sshConfig := &ssh.ClientConfig{
		User: username,
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
			command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4,
				"sudo", "nmap", "-oA", instance.System.NmapDir+"/"+timestamp+"_"+outputFile, "-p", port, additionOpts, ips)

			// //PING SCAN
			// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,username + "@" + ipv4,
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
		command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4,
			"sudo", "nmap", "-oA", instance.System.NmapDir+"/"+timestamp+"_"+outputFile, "-p", ports, additionOpts, ips)

		// //PING SCAN
		// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,username + "@" + ipv4,
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
		User: username,
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
