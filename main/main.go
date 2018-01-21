package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	// "strings"
)

//Cloud Proxy Tool
func main() {
	config := parseConfig()

	//Delete existing droplets
	droplets := listDroplets(config)
	destroyMultipleDroplets(config, droplets)

	//StartInstances
	allInstances, terminationMap := startInstances(config)
	config.AWS.Termination = terminationMap
	fmt.Println(config)
	if len(allInstances) == 0 {
		log.Fatal("No instances created. Check your shit bro...")
	}

	// ports := strings.Split("427,5631,13,873,5051,23,2717,5900,544,1025,53,25,8888,135,6001,119,9999,445,49157,5357,51326,8080,6646,2001,8008,199,514,8000,646,21,110", ",")
	// ports := strings.Split("49152,993,5432,515,2049,9,8081,8081,631,443,1723,4899,5009,9100,444,6000,5666,8009,32768,995,10000,1029,5190,3306,1110,22,88,7,554", ",")
	// ports := strings.Split("3389,179,587,79,5800,1900,2000,3128,465,3986,143,1720,389,3000,7070,5060,111,990,144,139,8443,5000,37,5101,2121,106,548,1433,543,113,1755", ",")
	//Just testing ports
	// ports := strings.Split("443", ",")

	//Not sure if this is the best way to go about it

	// fmt.Println(allInstances[0])

	// // //Gathering Information From Cloud Instances
	// allInstances = setHomeDirs(allInstances)

	//Setting Up Proxychains
	// _, proxychains, socksd := createMultipleSOCKS(allInstances, config.StartPort)

	//Setting Up Single SOCKS
	allInstances[0].createSingleSOCKS(8081)

	// // editProxychains(config.Proxychains, proxychains, 1)
	// fmt.Println("\n\nProxychains Configuration: ")
	// fmt.Println(proxychains)
	// fmt.Println("Wating to end:")
	// fmt.Println("\n\nSOCKSD Configuration: ")
	// fmt.Println(socksd)

	// //Running Nmap
	// runConnectScans(allInstances, "schein_connect_discovery", "–host-timeout 1m -Pn -sT -T2 --open", true, "/Users/mike.hodges/Gigs/HenrySchein/scope", ports)
	// go checkAllNmapProcesses(allInstances)

	//Teamserver Junk
	// allInstances[1].teamserverSetup(config, "ms.profile", "test", "2018-05-21")
	// fmt.Println("Now Back baby")

	//Check Cloudfront

	// //Create Cloudfront
	// createCloudFront(config, "testy mctest test", "www.example.com")

	// //Delete cloudfronts
	// for _, p := range listCloudFront(config) {
	// 	distribution,ETag := getCloudFront(*p.Id, config)
	// 	disableCloudFront(distribution, ETag, config)
	// }

	log.Println("Please CTRL-C to destroy droplets")

	// Catch CTRL-C and delete droplets.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// // editProxychains(config.Proxychains, proxychains, 0)
	allInstances = stopInstances(config, allInstances)
}

//Priorities:
// 1. Finish Google Domain Fronting Automation
// 2. Finish Security Groups and Firewalls for DO/AWS
// 3. Finish Cloudfronting
// 4. Add ability to stop/start EC2 instances
// 5. Add ability to import existing instances
// 6. Auto Drone-nmap on retrieval
// 7. Interface
// 8. Log all the things
// 9. Add more cloud providers
