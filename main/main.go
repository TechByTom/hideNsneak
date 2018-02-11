package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	// "strings"
	"github.com/rmikehodges/SneakyVulture/cloud"
)

//Cloud Proxy Tool
func main() {
	config := cloud.ParseConfig("../config/config.yaml")

	fmt.Println("destroying old droplets")
	cloud.DestroyAllDroplets(config.DO.Token)

	// StartInstances
	allInstances, terminationMap := cloud.StartInstances(config)
	config.AWS.Termination = terminationMap
	// fmt.Println(config)
	if len(allInstances) == 0 {
		log.Fatal("No instances created. Check your shit bro...")
	}

	//ports := strings.Split("427,5631,13,873,5051,23,2717,5900,544,1025,53,25,8888,135,6001,119,9999,445,49157,5357,51326,8080,6646,2001,8008,199,514,8000,646,21,110", ",")
	//ports := strings.Split("49152,993,5432,515,2049,9,8081,8081,631,443,1723,4899,5009,9100,444,6000,5666,8009,32768,995,10000,1029,5190,3306,1110,22,88,7,554", ",")
	// ports := strings.Split("3389,179,587,79,5800,1900,2000,3128,465,3986,143,1720,389,3000,7070,5060,111,990,144,139,8443,5000,37,5101,2121,106,548,1433,543,113,1755", ",")
	//Just testing ports
	// ports := strings.Split("80,443", ",")

	//Not sure if this is the best way to go about it

	// fmt.Println(allInstances[0])

	// // //Gathering Information From Cloud Instances
	cloud.Initialize(allInstances, config)

	//Setting Up Proxychains
	proxychains, socksd := cloud.CreateSOCKS(allInstances[:1], config.StartPort)

	fmt.Println(proxychains)
	fmt.Println(socksd)

	// // editProxychains(config.Proxychains, proxychains, 1)

	//fmt.Println("Running nmaps")
	// //Running Nmap
	// cloud.RunConnectScans(allInstances[1:], "schein_europe_connect_discovery", "-Pn -sT -T2 --open",
	// 	true, "/Users/mike.hodges/Gigs/HenrySchein/europe/scope.hosts", ports, config.NmapDir, false)

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

	log.Println("Please CTRL-C to destroy instances")

	// Catch CTRL-C and delete droplets.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// // editProxychains(config.Proxychains, proxychains, 0)
	cloud.StopInstances(config, allInstances)
}

//Priorities:
// 1. Interface
// 2. Log all the things
// 3. Add ability to import existing instances
// 4. Finish Cloudfronting

// 2. Finish Security Groups and Firewalls for DO/AWS
// 4. Add ability to stop/start EC2 instances
// 5.
