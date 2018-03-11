package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rmikehodges/hideNsneak/cloud"
	"github.com/rmikehodges/hideNsneak/sshext"
)

//Cloud Proxy Tool
func main() {
	// 	config := cloud.ParseConfig("../config/config.yaml")

	// 	fmt.Println("destroying old droplets")
	// 	cloud.DestroyAllDroplets(config.DO.Token)

	// 	// StartInstances
	// 	allInstances, terminationMap := cloud.StartInstances(config)
	// 	config.AWS.Termination = terminationMap
	// 	// fmt.Println(config)
	// 	if len(allInstances) == 0 {
	// 		log.Fatal("No instances created. Check your shit bro...")
	// 	}

	// 	//ports := strings.Split("427,5631,13,873,5051,23,2717,5900,544,1025,53,25,8888,135,6001,119,9999,445,49157,5357,51326,8080,6646,2001,8008,199,514,8000,646,21,110", ",")
	// 	//ports := strings.Split("49152,993,5432,515,2049,9,8081,8081,631,443,1723,4899,5009,9100,444,6000,5666,8009,32768,995,10000,1029,5190,3306,1110,22,88,7,554", ",")
	// 	//ports := strings.Split("3389,179,587,79,5800,1900,2000,3128,465,3986,143,1720,389,3000,7070,5060,111,990,144,139,8443,5000,37,5101,2121,106,548,1433,543,113,1755", ",")
	// 	//Just testing ports
	// 	//ports := strings.Split("80,443", ",")

	// 	//Not sure if this is the best way to go about it

	// 	// fmt.Println(allInstances[0])

	// 	// // //Gathering Information From Cloud Instances
	// 	cloud.Initialize(allInstances, config)

	// 	//Setting Up Proxychains
	// 	proxychains, socksd := cloud.CreateSOCKS(allInstances[:1], config.StartPort)

	// 	fmt.Println(proxychains)
	// 	fmt.Println(socksd)

	// 	// // editProxychains(config.Proxychains, proxychains, 1)

	// 	//fmt.Println("Running nmaps")
	// 	// //Running Nmap
	// 	// cloud.RunConnectScans(allInstances[1:], "schein_europe_connect_discovery", "-Pn -sT -T2 --open",
	// 	// 	true, "/Users/mike.hodges/Gigs/HenrySchein/europe/scope.hosts", ports, config.NmapDir, false)

	// 	//Teamserver Junk
	// 	// allInstances[1].teamserverSetup(config, "ms.profile", "test", "2018-05-21")
	// 	// fmt.Println("Now Back baby")

	// 	//Check Cloudfront

	// 	// //Create Cloudfront
	// 	// createCloudFront(config, "testy mctest test", "www.example.com")

	// 	// //Delete cloudfronts
	// 	// for _, p := range listCloudFront(config) {
	// 	// 	distribution,ETag := getCloudFront(*p.Id, config)
	// 	// 	disableCloudFront(distribution, ETag, config)
	// 	// }

	// 	log.Println("Please CTRL-C to destroy instances")

	// 	// Catch CTRL-C and delete droplets.
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// <-c

	// 	// // editProxychains(config.Proxychains, proxychains, 0)
	// 	cloud.StopInstances(config, allInstances)

	var allInstances []*Instance
	var terminationMap map[string][]string
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("<hideNsneak>")
		command, _ := reader.ReadString('\n')
		switch command {
		case "deployServers":
			freshDeploy, terminationMap := deployUI(config)
			allInstances = append(allInstances, freshDeploy)
			//TODO: Update termination map
		case "destroyServers":
		case "shell":
			shellUI(allInstances)
		case "list":
			listUI(allInstances)
		case "socks":
			socksUI(allInstances)
		case "domainFront":
		case "nmap":
			freshDeploy, terminationMap := deployUI(config)
			allInstances = append(allInstances, freshDeploy)
			nmapUI(freshDeploy)

			//TODO: Update Termination Map
			//Maybe just store termination codes in the instance itself
		case "proxyconf":
			fmt.Println("Proxychains:")
			fmt.Println(proxychains)
			fmt.Println("Socksd:")
			fmt.Println(socksd)
		}
	}

}

func nmapUI(instances []*cloud.Instance, config cloud.Config) {
	var scopeFile string
	var ports []string
	evasive := true
	reader := bufio.NewReader(os.Stdin)
	//Add ability to use exisiting servers
	//Gathering Ports and Port validation
	for {
		var portCheck int
		var err Error
		fmt.Print("<hideNSneak> Enter comma-seperated list of ports [ex. 80,443,8080-8082]")
		portString, _ := reader.ReadString('\n')
		portArray := strings.Split(portString,",")

		var tempPorts string
		for _, port := range portArray {
			if strings.Contains(port, "-") {
				portRange := strings.Split("-")
				startPort := strconv.Atoi(portRange[0])
				stopPort := strconv.Atoi(portRange[1])
				for i := startPort; i <= stopPort; i++ {
					tempPorts := tempPorts + strconv.Itoa(i) + ","
				}
			}
			tempPorts := tempPorts + port
		}

		tempPorts := tempPorts[:len(tempPorts)-1]
		portArray = strings.Split(tempPorts)

		for _, port := range portArray {
			portCheck, err = strconv.Atoi(port) 
			if err != nil | port > 65535 | port < 1 {
				break
			}
		}

		if err != nil {
			fmt.Println("<hideNSneak>Invalid Integer, one of your ports is not valid")
			continue
		}
		if port > 65535 | port < 1 {
			fmt.Println("<hideNSneak>Invalid Integer, one of your ports is not in the valid range 1-65535")
			continue
		}
		err = nil
		ports := portArray
		break
	}
	//Scope gathering and Scope file validation
	for {
		fmt.Print("<hideNSneak>Enter the file path for the scope file")
		scope, _ := reader.ReadString('\n')
		_, err := os.Stat(scope)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("<hideNSneak>File Error - Specified file does not exist")
				continue
			}
		}
		  file, err = os.OpenFile("test.txt", os.O_RDONLY, 0666)
		  if err != nil {
			  if os.IsPermission(err) {
				fmt.Println("<hideNSneak>File Error - Read access denied")
				continue
			  }
		  }
		  file.Close()
		  scopeFile := scope
		  break
	}
	fmt.Print("<hideNSneak>Enter the base name for the nmap files: ")
	baseName, _ := reader.ReadString('\n')
	for {
		fmt.Print("<hideNSneak>Enter the additional Nmap options needed: ")
		additionalOpts, _ := reader.ReadString('\n')
		fmt.Println("<hideNSneak>Your resulting command:")
		fmt.Println("<hideNSneak>nmap -iL " + scopeFile + "-oA" + basename + "-<timestamp>" + additionalOpts + "-p <ports> <ips>" +)
		for {
			fmt.Print("<hideNSneak> Is this correct? [Y/N/Q]: ")
			correctNmap, _ := reader.ReadString('\n')
			switch strings.UpperCase(correctNmap) {
			case "Y":
				break
			case "N":
				continue
			case "Q":
				return
			default:
				fmt.Println("<hideNSneak> Please answer Y or N")
				continue
			}
		}
		break
	}
	fmt.Print("<hideNSneak> Would you Nmap to be evasive? [Y/N]: ")
	evasiveResponse, _ = reader.ReadString('\n')
	if strings.UpperCase(evasiveResponse) := "N" {
		evasive = false
	}
	//TODO: Add automatic drone-nmap
	cloud.RunConnectScans(cloud.RunConnectScans(instances, baseName,additionalOpts, evasive,scopeFile,
		ports, config.NmapDir, false))
}


func deployUI(config cloud.Config) ([]*cloud.Instance, map[string][]string) {
	reader := bufio.NewReader(os.Stdin)
	var count int
	var providerArray []string
	for {
		fmt.Print("<hideNsneak> Enter the cloud providers you would like to use [Default: AWS,DO]: ")
		providers, _ := reader.ReadString('\n')
		if providers == "" {
			providerArray = strings.Split("AWS,DO", ",")
		} else {
			providerArray = strings.Split(providers, ",")
			for _, p := range providerArray {
				if strings.ToUpper(p) != "AWS" && strings.ToUpper(p) != "DO" {
					fmt.Println("Unknown Cloud Provider, please check your input..")
					continue
				}
			}
		}
		break
	}
	for {
		fmt.Print("<hideNSneak> Enter the number of servers to deploy: ")
		countString, _ := reader.ReadString('\n')
		count, err := strconv.Atoi(countString)
		if err != nil {
			fmt.Println("<hideNSneak> Err: Not an Integer  ")
			continue
		}
		break
	}

	providerMap := make(map[string]int)
	division := count / len(providers)
	remainder := count % len(providers)

	for _, p := range providers {
		providerMap[p] := division
	}

	for p := range providerMap {
		if remainder == 0 {
			break
		}
		providerMap[p] == providerMap[p] + 1
		remainder = remainder - 1
	}


	instanceArray, terminationMap := cloud.StartInstances(config, providerMap)

	cloud.Initialize(instanceArray, config)

	return instanceArray, terminationMap
}

func listUI(instances []*cloud.Instance) {
	for num, instance := range instances {
		fmt.Print(num + " : ")
		instance.String()
	}
}

func shellUI(instances []*cloud.Instance) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("<hideNSneak> Servers:")
	listUI(instances)
	fmt.Print("<hideNSneak> Enter the number of the server you wish drop into")
	instanceNum, _ := reader.ReadString('\n')
	num, err := strconv.Atoi(instanceNum)
	if err != nil {
		fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
	}
	sshext.ShellSystem( instances[num].Cloud.IPv4,instances[num].SSH.Username, instances[num].SSH.PrivateKey)
}


func socksUI(instances []*cloud.Instance) (string, string) {
	var proxychains string
	var socksd string
	reader := bufio.NewReader(os.Stdin)
	startPort := "8081"
	fmt.Println("<hideNSneak> Servers:")
	listUI(instances)
	for {
		fmt.Print("<hideNSneak> Enter the start port for your SOCKS proxies [Default: 8081]: ")
		stringPort, _ := reader.ReadString('\n')
		_, err := strconv.Atoi(stringPort)
		if err != nil {
			fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
		}
	}
	fmt.Print("<hideNSneak> Enter a comma seperated list of servers to create SOCKS proxies [Default: all]")
	instanceString, _ := reader.ReadString('\n')
	if instanceString == "" {
		proxychains, socksd = cloud.CreateSOCKS(instances)
	} else {
		instanceArray := strings.Split(instanceString ,",")
		tempInstances := []*cloud.Instance
		for _, p := range instanceArray {
			tempInstances := append(tempInstances, instances[p])
		}
		proxychains, socksd = cloud.CreateSOCKS(tempInstances)
	}
	return proxychains, socksd
}

//Priorities:
// 1. Interface
// 2. Log all the things
// 3. Add ability to import existing instances
// 4. Finish Cloudfronting

//I'm going to have to a abstract the logging away from the cloud package

// 2. Finish Security Groups and Firewalls for DO/AWS
// 4. Add ability to stop/start EC2 instances
// 5.
