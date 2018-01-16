package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
	"os/signal"
	"gopkg.in/yaml.v2"
	// "strings"
)

//Cloud Proxy Tool

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func removeString(s []string, e string) []string {
	for i := range s {
		if s[i] == e {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}


type CloudInstance struct {
	Type        string
	Description string
	ID          string
	Region      string
	Username    string
	IPv4        string
	SOCKSActive bool
	SOCKSPort   string
	PrivateKey  string
	CMD         *exec.Cmd
	Stderr      *bufio.Reader
	HomeDir		string
	NmapDir		string
	NmapTargets map[string][]string
	NmapActive bool
	NmapCmd string
	NmapProcess string
	Firewalls	[]string
	Config Config
}

type Config struct {
	PublicKey   string `yaml:"PublicKey"`
	Customer	string `yaml:"Customer"`
	StartPort   int    `yaml:"StartPort"`
	Proxychains string `yaml:"Proxychains"`
	CSDir string `yaml:"CobaltStrikeDir"`
	CSProfiles 	string `yaml:"CobaltStrikeProfiles"`
	CSLicense string `yaml:"CobaltStrikeLicense"`
	NmapDir string `yaml:"NmapDir"`
	AWS         struct {
		Secret      string `yaml:"secret"`
		AccessID    string `yaml:"accessID"`
		Regions     string `yaml:"regions"`
		ImageIDs    string `yaml:"imageIDs"`
		Type        string `yaml:"type"`
		Number      int    `yaml:"number"`
		Termination map[string][]string
		SecurityGroups map[string][]string
	} `yaml:"AWS"`
	DO struct {
		Token       string `yaml:"token"`
		Fingerprint string `yaml:"fingerprint"`
		Regions     string `yaml:"regions"`
		Slug        string `yaml:"slug"`
		Memory      string `yaml:"memory"`
		Name        string `yaml:"name"`
		Number      int    `yaml:"number"`
	} `yaml:"DO"`
}

func (config *Config) updateTermination(terminationMap map[string][]string) {
	config.AWS.Termination = terminationMap
}

func removeDuplicateStrings(inSlice []string) (outSlice []string){
	fmt.Println(inSlice)
	outSlice = inSlice[:1]
	for _,p:= range inSlice {
		inOutSlice := false
		for _,q := range outSlice {
			if p == q {
				inOutSlice = true
			}
		}
		if !inOutSlice {
			outSlice = append(outSlice, p)
		}
	}
	return
}


func parseConfig() Config {
	var config Config
	data, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func combineToMap(allInstances []CloudInstance) map[int]*CloudInstance {
	instanceMap := make(map[int]*CloudInstance)
	for i := range allInstances {
		instanceMap[i] = &allInstances[i]
	}
	return instanceMap
}

func stopInstances(config Config, allInstances map[int]*CloudInstance) map[int]*CloudInstance{
	for _, instance := range allInstances {
		if instance.Type == "DO" {
			destroyDOInstance(*instance)
		}
	}
	fmt.Println("About to terminate")
	terminateEC2Instances(config.AWS.Termination, config.AWS.Secret, config.AWS.AccessID)
	for p := range allInstances {
		if allInstances[p].SOCKSActive == true && allInstances[p].CMD.Process != nil {
			error := allInstances[p].CMD.Process.Kill()
			allInstances[p].SOCKSActive = false
			if error != nil {
				fmt.Println("Error killing socks process")
				fmt.Println(error)
			}
		}
	}
	return allInstances
}

func getIPAddresses(allInstances map[int]*CloudInstance, config Config) {
	for k := range allInstances {
		if allInstances[k].Type == "EC2" {
			allInstances[k].IPv4 = getEC2IP(allInstances[k].Region, config.AWS.Secret, config.AWS.AccessID, allInstances[k].ID)
		}
		if allInstances[k].Type == "DO" {
			doID, _ := strconv.Atoi(allInstances[k].ID)
			allInstances[k].IPv4 = getDOIP(config.DO.Token, doID)
		}
	}
	// return allInstances
}

func startInstances(config Config) (map[int]*CloudInstance, map[string][]string) {
	ec2Result := 0
	doResult := 0
	var terminationMap map[string][]string
	var ec2Instances []CloudInstance
	var doInstances []CloudInstance
	var allInstances []CloudInstance
	var mappedInstances map[int]*CloudInstance
	if config.AWS.Number > 0 {
		ec2Instances, ec2Result, terminationMap = deployMultipleEC2(config)
		fmt.Println(ec2Instances)
		allInstances = append(allInstances, ec2Instances...)
	}
	if config.DO.Number > 0 {
		doInstances, doResult = deployDO(config)
		allInstances = append(allInstances, doInstances...)
	}
	if config.AWS.Number > 0 || config.DO.Number > 0 {
		mappedInstances = combineToMap(allInstances)
		if ec2Result == 1 || doResult == 1 {
			stopInstances(config, mappedInstances)
		}
		fmt.Println("Waiting a few seconds for all instances to initialize...")
		time.Sleep(60 * time.Second)
		getIPAddresses(mappedInstances, config)
		for p := range mappedInstances {
			fmt.Println(mappedInstances[p])
		}
	}
	return mappedInstances, terminationMap
}


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

//TODO
//1.  Implement Ability to import existing machines into cloudInstances
//2.  Add Support for LetsEncrypt
//3.  Add Domain Fronting (CloudFront)
//8.  Add API gateway support
//9.  Add centralized Log support - BIG
//10. Add ability to stop EC2 instances to preserve data
//11. Add ability to connect back to started machines that weren't terminated
//12. Add automatic drone-nmap functionality
//		--if-error, save file name, continue, and then redo the errors
//13. Add ability scale back nmap scans and queue them for later, bring down hosts, and stand new ones up for scanning


//NOTES
//1. Partially implemented functionality for EC2 security groups
//2. Implement evasice scanning method that will scan a portion of the ports --> bring down the host --> resume scanning with a new host
//3. By default, teamservers are deployed to EC2 instances as they can be stopped without losing their storage, thus preserving important artifacts