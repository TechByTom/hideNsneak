package nmap

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rmikehodges/hideNsneak/misc"
	"github.com/rmikehodges/hideNsneak/sshext"
	"golang.org/x/crypto/ssh"
)

func InitiateConnectScan(username string, ipv4 string, privateKey string, nmapTargets map[int][]string,
	homedir string, localDir string, additionOpts string, evasive bool, region string, instanceType string) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			sshext.PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	fmt.Println("Installing nmap")
	sshext.ExecuteCmd(`sudo apt-get update;
		sudo apt-get install -y nmap;
		sudo apt-get install -y screen`, ipv4, sshConfig)

	fmt.Println("Successfully installed Nmap")

	nmapDir := homedir + "/" + ipv4 + "-nmap"
	fmt.Println("Making directory")
	sshext.ExecuteCmd("mkdir "+nmapDir, ipv4, sshConfig)
	if evasive {
		for port, ipList := range nmapTargets {
			portString := strconv.Itoa(port)
			fmt.Println("In the evasive scanning if statement")
			timestamp := time.Now().Format("20060102150405")
			ips := (strings.Join(ipList, " "))
			nmapCommand := fmt.Sprintf("nmap -oA"+" "+nmapDir+"/"+timestamp+"_"+portString+" "+"-p%d"+" "+additionOpts+" "+ips, port)

			fmt.Println("Executing nmap")
			fmt.Println(nmapCommand)

			// PORT SC

			command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4,
				"sudo", "nmap", "-oA", nmapDir+"/"+timestamp+"_"+portString, "-p", strconv.Itoa(port), additionOpts, ips)

			misc.WriteActivityLog(instanceType + " " + ipv4 + " " + region + " Disptaching nmap scan to host : " + nmapCommand)

			// //PING SCAN
			// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,username + "@" + ipv4,
			// 	"sudo", "nmap", "-oA", instance.Nmap.NmapLocalDir + "/" + timestamp + "_" + outputFile, additionOpts, ips  )

			//Cmd Exec run is consuming a lot of memory due to the fact the method must hold.
			if err := command.Run(); err != nil {
				misc.WriteActivityLog(instanceType + " " + ipv4 + " " + region + " There was an error running the nmap scan - see error log")
				misc.WriteErrorLog(instanceType + " " + ipv4 + " " + region + " Error while dispatching scan  : " + fmt.Sprint(err))
				return
			}
		}
	} else {
		// 	var portList []string
		// 	var finalIPList []string
		// 	for port, ipList := range instance.Nmap.NmapTargets {
		// 		portList = append(portList, port)
		// 		finalIPList = append(finalIPList, ipList...)
		// 	}
		// 	portList = removeDuplicateStrings(portList)
		// 	timestamp := time.Now().Format("20060102150405")
		// 	ports := strings.Join(portList, ",")
		// 	ips := strings.Join(finalIPList, " ")
		// 	instance.Nmap.NmapCmd = "nmap -oA " + instance.Nmap.NmapLocalDir + "/" + timestamp + "_" + outputFile + " -p" + ports + " " + additionOpts + " " + ips

		// 	fmt.Println("Executing nmap")
		// 	fmt.Println(instance.Nmap.NmapCmd)

		// 	// PORT SCAN
		// 	command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4,
		// 		"sudo", "nmap", "-oA", instance.Nmap.NmapLocalDir+"/"+timestamp+"_"+outputFile, "-p", ports, additionOpts, ips)

		// 	// //PING SCAN
		// 	// command := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey,username + "@" + ipv4,
		// 	// 	"sudo", "nmap", "-oA", instance.Nmap.NmapLocalDir + "/" + timestamp + "_" + outputFile, additionOpts, ips  )
		// 	instance.Nmap.NmapActive = true
		// 	if err := command.Run(); err != nil {
		// 		fmt.Println("nmap")
		// 		fmt.Println(err)
		// 	}
		// 	instance.Nmap.NmapActive = false
		// }

	}
	if !sshext.RsyncDirFromHost(nmapDir, localDir, username, ipv4, privateKey) {
		fmt.Println("done")
		misc.WriteActivityLog(instanceType + " " + ipv4 + " " + region + " unable to rsync nmap files")
		return
	}
	misc.WriteActivityLog(instanceType + " " + ipv4 + " " + region + " : rsync'd files from host")
	return
}

func ListNmapXML(nmapDir string) []string {
	var outb, errb bytes.Buffer
	cmd := exec.Command("find", nmapDir, "-type", "f", "-name", "*.xml")
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	output := outb.String()
	if err != nil {
		fmt.Println("Problem running find")
		return nil
	}
	return strings.Split(output, "\n")
}

func CheckNmapProcess(ipv4 string, username string, privateKey string, nmapCmd string) (string, bool) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			sshext.PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	grepString := "ps aux | grep '" + nmapCmd + "' | grep -v grep | awk '{print $2}'"

	process := sshext.ExecuteCmd(grepString, ipv4, sshConfig)

	proccessList := strings.Split(process, "\n")

	processString := strings.Join(proccessList[:len(proccessList)-1], ",")
	if len(processString) < 4 {
		return "", false
	} else {
		return processString, true
	}
}

//NOTES: Max parallel host size to 1 for more resillient scans based on time window
