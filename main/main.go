package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rmikehodges/hideNsneak/cloud"
	"github.com/rmikehodges/hideNsneak/misc"
	"github.com/rmikehodges/hideNsneak/sshext"
)

//Cloud Proxy Tool
func main() {
	config := cloud.ParseConfig("../config/config.yaml")

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

	var allInstances []*cloud.Instance
	var allDomainFronts []cloud.DomainFront
	// var terminationMap map[string][]string
	var proxychains, socksd string
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("<hideNsneak> ")
		command, _ := reader.ReadString('\n')
		switch strings.TrimSpace(command) {
		case "deploy":
			//Deployment Procedure
			reader := bufio.NewReader(os.Stdin)
			var providerArray []string
			var providers string
			var count int
			var err error
			for {
				fmt.Print("<hideNsneak> Enter the cloud providers you would like to use [Default: AWS,DO, Google]: ")
				providers, _ = reader.ReadString('\n')
				providers = strings.TrimSpace(providers)
				if providers == "" {
					providerArray = []string{"AWS", "DO", "Google"}
				} else {
					providerArray = strings.Split(providers, ",")
					if providerCheck(providerArray) {
						break
					}
				}
			}
			for {
				fmt.Print("<hideNSneak> Enter the number of servers to deploy: ")
				countString, _ := reader.ReadString('\n')
				countString = strings.TrimSpace(countString)
				count, err = strconv.Atoi(countString)
				if err != nil {
					fmt.Println("<hideNSneak> Err: Not an Integer  ")
					continue
				}
				break
			}
			providerMap := make(map[string]int)
			division := count / len(providerArray)
			remainder := count % len(providerArray)

			for _, p := range providerArray {
				providerMap[p] = division
			}

			if remainder != 0 {
				for p := range providerMap {
					providerMap[p] = providerMap[p] + 1
					remainder = remainder - 1
					if remainder == 0 {
						break
					}
				}
			}

			instanceArray := cloud.StartInstances(config, providerMap)
			allInstances = append(allInstances, instanceArray...)
			//TODO: Update termination map
		case "destroy":
			//TODO Fix destroy to correct list remaining servers
			reader := bufio.NewReader(os.Stdin)
			tempInstances := []*cloud.Instance{}
			var instanceArray []string
			listUI(allInstances)

			//newInstanceList := []*cloud.Instance{}

			for {
				fmt.Println("<hideNSneak> Enter a comma separated list of servers to destroy [Default: all]")
				instanceString, _ := reader.ReadString('\n')
				instanceString = strings.TrimSpace(instanceString)

				instanceArray = strings.Split(instanceString, ",")

				if misc.ValidateIntArray(instanceArray) == false && instanceArray[0] != "" {
					fmt.Println("<hideNSneak> Server specification contains non-integers, try again")
					continue
				}
				break
			}
			//Creating List for destruction
			if instanceArray[0] != "" {
				fmt.Println(instanceArray)
				for _, p := range instanceArray {
					fmt.Println(p)
					index, _ := strconv.Atoi(p)

					if index >= len(allInstances) {
						fmt.Println("<hideNSneak> Index is larger than the instance array. Skipping index: " + strconv.Itoa(index))
						continue
					}

					tempInstances = append(tempInstances, allInstances[index])
				}
			} else {
				tempInstances = allInstances
			}
			for {
				fmt.Println("<hideNSneak> The following servers will be deleted - Is this ok [y/n]")
				listUI(tempInstances)
				confirmation, _ := reader.ReadString('\n')
				confirmation = strings.TrimSpace(confirmation)
				if strings.ToLower(string(confirmation[0])) == "y" {
					cloud.DestroyInstances(config, tempInstances)
					for _, p := range instanceArray {
						index, _ := strconv.Atoi(p)
						if index < len(allInstances)-1 {
							if index == 0 && len(allInstances) == 1 {
								fmt.Println("")
								allInstances = []*cloud.Instance{}
								break
							}
							if index == 0 {
								allInstances = allInstances[1:]
							} else {
								allInstances = append(allInstances[:index], allInstances[index+1:]...)
							}
						} else {
							//todo: this is where its fuckin up
							allInstances = allInstances[:index]
						}
					}
					// allInstances = newInstanceList
					break
				}
				if strings.ToLower(string(confirmation[0])) == "n" {
					break
				}
			}

		case "shell":
			reader := bufio.NewReader(os.Stdin)
			listUI(allInstances)
			fmt.Println("<hideNSneak> Enter the number of the server you wish drop into: ")
			instanceNum, _ := reader.ReadString('\n')
			instanceNum = strings.TrimSpace(instanceNum)
			num, err := strconv.Atoi(strings.TrimSpace(instanceNum))
			if err != nil {
				fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
			}
			if num > len(allInstances) {
				fmt.Println("<hideNSneak> That instance does not exist - try spinning some up")
				return
			}
			sshext.ShellSystem(allInstances[num].Cloud.IPv4, allInstances[num].SSH.Username, allInstances[num].SSH.PrivateKey)
		case "list":
			//TODO Add Ability to specify provider
			listUI(allInstances)
		case "socksAdd":
			startPort := config.StartPort
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("<hideNSneak> Servers:")
			listUI(allInstances)
			for {
				fmt.Print("<hideNSneak> Enter the start port for your SOCKS proxies [Default: Config Value]: ")
				stringPort, _ := reader.ReadString('\n')
				stringPort = strings.TrimSpace(stringPort)
				if stringPort == "" {
					break
				}
				_, err := strconv.Atoi(stringPort)
				if err != nil {
					fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
					continue
				}
				break
			}
			fmt.Println("<hideNSneak> Enter a comma seperated list of servers to create SOCKS proxies [Default: all]")
			instanceString, _ := reader.ReadString('\n')
			instanceString = strings.TrimSpace(instanceString)
			if instanceString == "" {
				proxychains, socksd = cloud.CreateSOCKS(allInstances, startPort)
				config.StartPort = config.StartPort + len(allInstances)
			} else {
				instanceArray := strings.Split(instanceString, ",")
				tempInstances := []*cloud.Instance{}
				for _, p := range instanceArray {
					index, _ := strconv.Atoi(p)
					tempInstances = append(tempInstances, allInstances[index])
				}
				proxychains, socksd = cloud.CreateSOCKS(tempInstances, config.StartPort)
				config.StartPort = config.StartPort + len(tempInstances)
			}
		case "socksRemove":
		//TODO Add Socks-remove functionality
		case "domainFront-create":
			reader := bufio.NewReader(os.Stdin)
			for {
				fmt.Print("<hideNsneak> Enter the cloud provider you would like to use [Options: Google, AWS]: ")
				provider, _ := reader.ReadString('\n')
				provider = strings.TrimSpace(provider)
				if provider == "" {
					continue
				} else {
					if provider == "AWS" {
						fmt.Print("<hideNsneak> Enter the domain name you want your cloudfront distro to point to: ")
						domain, _ := reader.ReadString('\n')
						domain = strings.TrimSpace(domain)

						fmt.Print("<hideNsneak> Is this correct? [Y/n]: " + domain)
						confirmation, _ := reader.ReadString('\n')
						if strings.ToLower(string(confirmation[0])) == "n" {
							break
						}
						if strings.ToLower(string(confirmation[0])) == "y" {
							cloudFrontCreation := cloud.CreateCloudfront(config, domain)
							if cloudFrontCreation.Type != "" {
								allDomainFronts = append(allDomainFronts, cloudFrontCreation)
								cloudFrontCreation.Target = domain
								fmt.Println("<hideNsneak> Cloudfront distribution created:")
								fmt.Println("<hideNsneak> " + cloudFrontCreation.Host + "-->" + cloudFrontCreation.Target)
							} else {
								fmt.Println("<hideNsneak> Cloudfront distrubtion not created properly, please check the error log")
							}
						}
						break
					}
					if provider == "Google" {
						//TODO: Ensure that the newlines are being properly parsed out
						var userAgent string
						var subnet string
						var header string

						var redirection = "https://google.com"

						var keystore string
						var keystorePass string

						var newProject = false

						fmt.Print("<hideNsneak> Enter the domain name/IP address you want your cloudfront distro to point to: ")
						domain, _ := reader.ReadString('\n')
						domain = strings.TrimSpace(domain)
						fmt.Print("<hideNsneak> Would you like to use HTTPS? [y/n]: ")
						https, _ := reader.ReadString('\n')
						https = strings.TrimSpace(https)
						if strings.ToLower(string(https[0])) == "y" {
							domain = "https://" + domain
							fmt.Print("<hideNsneak> Please enter the name of your Java Keystore for Cobalt Strike [Leave blank if N/A]: ")
							temp, _ := reader.ReadString('\n')
							keystore = strings.TrimSuffix(temp, "\n")

							fmt.Print("<hideNsneak> Please enter the password for your Java Keystore for Cobalt Strike [Leave blank if N/A]: ")
							temp, _ = reader.ReadString('\n')
							keystorePass = strings.TrimSuffix(temp, "\n")

						} else {
							domain = "http://" + domain
						}
						//List possible C2 profiles

						//TODO: Add restriction based on header

						//Restriction Based on User Agent
						fmt.Print("<hideNsneak> Would you like to restrict access based on User Agent? [y/n]: ")
						uaConfirmation, _ := reader.ReadString('\n')
						uaConfirmation = strings.TrimSpace(uaConfirmation)
						if strings.ToLower(string(uaConfirmation[0])) == "y" {
							fmt.Print("<hideNsneak> Enter the user agent you would like to restrict on: ")
							temp, _ := reader.ReadString('\n')
							userAgent = strings.TrimSuffix(temp, "\n")
						}

						//Restriction Based on Subnet
						fmt.Print("<hideNsneak> Would you like to restrict access based on Subnet?: ")
						subnetConfirmation, _ := reader.ReadString('\n')
						subnetConfirmation = strings.TrimSpace(subnetConfirmation)
						if strings.ToLower(string(subnetConfirmation[0])) == "y" {
							fmt.Print("<hideNsneak> Enter the subnet you would like to restrict access to: ")
							temp, _ := reader.ReadString('\n')
							subnet = strings.TrimSuffix(temp, "\n")
						}

						if strings.ToLower(string(subnetConfirmation[0])) == "y" || strings.ToLower(string(uaConfirmation[0])) == "y" {
							fmt.Print("<hideNsneak> What is the default redirect you would like to use for restrictions? [default: https://google.com]: ")
							temp, _ := reader.ReadString('\n')
							redirection = strings.TrimSuffix(temp, "\n")
						}

						//Checking if the project is new
						fmt.Print("<hideNsneak> Is this a new gcloud project? [y/n]: ")
						projectConfirmation, _ := reader.ReadString('\n')
						projectConfirmation = strings.TrimSpace(projectConfirmation)
						if strings.ToLower(string(projectConfirmation[0])) == "y" {
							newProject = true
						}

						//Add creation of C2 profiles
						result := cloud.CreateGoogleDomainFront(config, domain, keystore, keystorePass, newProject, userAgent,
							subnet, header, redirection, "")
						if result != "" {
							googleDomainFront := cloud.DomainFront{
								Type:   "Google",
								Host:   result,
								Target: domain,
							}
							allDomainFronts = append(allDomainFronts, googleDomainFront)
						}

					}
					break
				}
			}
		case "nmap":
			//TODO Test Nmap
			//TODO Add non-evasive scanning
			//TODO: Fix port Validation
			// freshDeploy := deployUI(config)

			//Deployment Procedure
			reader := bufio.NewReader(os.Stdin)
			var providerArray []string
			var providers string
			var count int
			var err error
			for {
				fmt.Print("<hideNsneak> Enter the cloud providers you would like to use [Default: AWS,DO]: ")
				providers, _ = reader.ReadString('\n')
				providers = strings.TrimSpace(providers)
				if providers == "" {
					providerArray = strings.Split("AWS,DO", ",")
				} else {
					providerArray = strings.Split(providers, ",")
					if providerCheck(providerArray) {
						fmt.Println("HERE")
						break
					}
				}
			}
			for {
				fmt.Print("<hideNSneak> Enter the number of servers to deploy: ")
				countString, _ := reader.ReadString('\n')
				countString = strings.TrimSpace(countString)
				count, err = strconv.Atoi(countString)
				if err != nil {
					fmt.Println("<hideNSneak> Err: Not an Integer  ")
					continue
				}
				break
			}
			providerMap := make(map[string]int)
			division := count / len(providerArray)
			remainder := count % len(providerArray)

			for _, p := range providerArray {
				providerMap[p] = division
			}

			if remainder != 0 {
				for p := range providerMap {
					providerMap[p] = providerMap[p] + 1
					remainder = remainder - 1
					if remainder == 0 {
						break
					}
				}
			}

			instanceArray := cloud.StartInstances(config, providerMap)
			allInstances = append(allInstances, instanceArray...)

			// nmapUI(freshDeploy, config)
			var scopeFile string
			var ports []string
			evasive := true
			reader = bufio.NewReader(os.Stdin)
			//Add ability to use exisiting servers
			//Gathering Ports and Port validation
			for {
				// var portCheck int
				var err error
				fmt.Print("<hideNSneak> Enter comma-seperated list of ports [ex. 80,443,8080-8082]")
				portString, _ := reader.ReadString('\n')
				portArray := strings.Split(portString, ",")

				var tempPorts string
				for _, port := range portArray {
					if strings.Contains(port, "-") {
						portRange := strings.Split(port, "-")
						startPort, _ := strconv.Atoi(portRange[0])
						stopPort, _ := strconv.Atoi(portRange[1])
						for i := startPort; i <= stopPort; i++ {
							tempPorts = tempPorts + strconv.Itoa(i) + ","
						}
					}
					tempPorts = tempPorts + port
				}

				tempPorts = tempPorts[:len(tempPorts)-1]
				portArray = strings.Split(tempPorts, ",")

				// for _, port := range portArray {
				// 	portCheck, err = strconv.Atoi(port)
				// 	if err != nil | portCheck > 65535 | portCheck < 1 {
				// 		break
				// 	}
				// }

				if err != nil {
					fmt.Println("<hideNSneak>Invalid Integer, one of your ports is not valid")
					continue
				}
				// if port > 65535 | port < 1 {
				// 	fmt.Println("<hideNSneak>Invalid Integer, one of your ports is not in the valid range 1-65535")
				// 	continue
				// }
				err = nil
				// ports := portArray
				break
			}
			//Scope gathering and Scope file validation
			for {
				fmt.Print("<hideNSneak>Enter the file path for the scope file")
				scope, _ := reader.ReadString('\n')
				scope = strings.TrimSpace(scope)
				_, err := os.Stat(scope)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Println("<hideNSneak>File Error - Specified file does not exist")
						continue
					}
				}
				file, err := os.OpenFile("test.txt", os.O_RDONLY, 0666)
				if err != nil {
					if os.IsPermission(err) {
						fmt.Println("<hideNSneak>File Error - Read access denied")
						continue
					}
				}
				file.Close()
				// scopeFile := scope
				break
			}
			fmt.Print("<hideNSneak>Enter the base name for the nmap files: ")
			baseName, _ := reader.ReadString('\n')
			baseName = strings.TrimSpace(baseName)
			var additionalOpts string
			for {
				fmt.Print("<hideNSneak>Enter the additional Nmap options needed: ")
				additionalOpts, _ = reader.ReadString('\n')
				fmt.Println("<hideNSneak>Your resulting command:")
				fmt.Println("<hideNSneak>nmap -iL " + scopeFile + "-oA" + baseName + "-<timestamp>" + additionalOpts + "-p <ports> <ips>")
				for {
					fmt.Print("<hideNSneak> Is this correct? [Y/N/Q]: ")
					correctNmap, _ := reader.ReadString('\n')
					switch strings.ToUpper(correctNmap) {
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
			evasiveResponse, _ := reader.ReadString('\n')
			if strings.ToUpper(evasiveResponse) == "N" {
				evasive = false
			}
			//TODO: Add automatic drone-nmap
			cloud.RunConnectScans(instanceArray, baseName, additionalOpts, evasive, scopeFile,
				ports, config.NmapDir, false)
			//TODO: Update Termination Map
		case "proxyconf":
			fmt.Println("Proxychains:")
			fmt.Println(proxychains)
			fmt.Println("Socksd:")
			fmt.Println(socksd)
		case "senddir":
			var err error
			reader := bufio.NewReader(os.Stdin)
			var originFilePath string
			var targetFilePath string
			var chosenServer int

			for {
				fmt.Println("<hideNSneak> Choose the remote server to send directory to: ")
				listUI(allInstances)
				chosenServerString, _ := reader.ReadString('\n')
				chosenServerString = strings.TrimSpace(chosenServerString)
				chosenServer, err = strconv.Atoi(chosenServerString)
				if err != nil {
					fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
					continue
				}
				if chosenServer > len(allInstances)-1 {
					fmt.Println("<hideNSneak> That instance does not exist - try spinning some up or try again")
					continue
				}
				break
			}
			ipV4 := allInstances[chosenServer].Cloud.IPv4
			userName := allInstances[chosenServer].SSH.Username
			privateKey := allInstances[chosenServer].SSH.PrivateKey

			for {
				fmt.Println("<hideNSneak> Enter filepath of local directory to send: ")
				originFilePath, err = reader.ReadString('\n')
				originFilePath = strings.TrimSpace(originFilePath)
				doesFileExist, err := misc.Exists(originFilePath)

				if err != nil {
					fmt.Println("<hideNSneak> Invalid filepath - Please check your input")
					continue
				}

				if doesFileExist == false {
					fmt.Println("<hideNSneak> Filepath doesn't exist - Please check your input")
					continue
				}
				break
			}

			for {
				fmt.Println("<hideNSneak> Enter filepath of target directory: ")
				targetFilePath, err = reader.ReadString('\n')
				targetFilePath = strings.TrimSpace(targetFilePath)

				if err != nil {
					fmt.Println("<hideNSneak> Invalid filepath - Please check your input")
					continue
				}
				break
			}

			sshext.RsyncDirToHost(originFilePath, targetFilePath, userName, ipV4, privateKey)

		case "getdir":
			var err error
			reader := bufio.NewReader(os.Stdin)
			var originFilePath string
			var targetFilePath string
			var chosenServer int

			for {
				fmt.Println("<hideNSneak> Choose the remote server to receive directory from: ")
				listUI(allInstances)
				chosenServerString, _ := reader.ReadString('\n')
				chosenServerString = strings.TrimSpace(chosenServerString)
				chosenServer, err = strconv.Atoi(chosenServerString)
				if err != nil {
					fmt.Println("<hideNSneak> Invalid Integer - Please check your input")
					continue
				}
				if chosenServer > len(allInstances)-1 {
					fmt.Println("<hideNSneak> That instance does not exist - try spinning some up or try again")
					continue
				}
				break
			}
			ipV4 := allInstances[chosenServer].Cloud.IPv4
			userName := allInstances[chosenServer].SSH.Username
			privateKey := allInstances[chosenServer].SSH.PrivateKey

			for {
				fmt.Println("<hideNSneak> Enter filepath of local directory to send: ")
				originFilePath, err = reader.ReadString('\n')
				originFilePath = strings.TrimSpace(originFilePath)

				if err != nil {
					fmt.Println("<hideNSneak> Invalid filepath - Please check your input")
					continue
				}
				break
			}

			for {
				fmt.Println("<hideNSneak> Enter filepath of target directory: ")
				targetFilePath, err = reader.ReadString('\n')
				targetFilePath = strings.TrimSpace(targetFilePath)

				if err != nil {
					fmt.Println("<hideNSneak> Invalid filepath - Please check your input")
					continue
				}
				break
			}

			sshext.RsyncDirFromHost(originFilePath, targetFilePath, userName, ipV4, privateKey)

		case "sendFile":
		//TODO add sendFile command
		case "getFile":
			//TODO add getFile command
		case "firewall":
		//TODO add firewall support
		case "quit":
			fmt.Println("<hideNsneak> Shutting Down")
			os.Exit(1)
		case "exit":
			fmt.Println("<hideNsneak> Shutting Down")
			os.Exit(1)
		default:
			fmt.Println("I stupid")
		}
	}

}

func listUI(instances []*cloud.Instance) {
	for num, instance := range instances {
		fmt.Printf("%d : %s \n", num, instance.String())
	}
}

func providerCheck(providerArray []string) bool {
	for _, p := range providerArray {
		if strings.ToUpper(p) != "AWS" && strings.ToUpper(p) != "DO" && strings.ToUpper(p) != "GOOGLE" {
			fmt.Println("Unknown Cloud Provider, please check your input..")
			return false
		}
	}
	return true
}

// 2. Finish Security Groups and Firewalls for DO/AWS

//TODO: Add instance import
//TODO: Add Stop Instance functionality on AWS,DO,Google
