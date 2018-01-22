package cloud

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/digitalocean/godo"
	"github.com/rmikehodges/SneakyVulture/amazon"
	"github.com/rmikehodges/SneakyVulture/do"
	"github.com/rmikehodges/SneakyVulture/nmap"
	"github.com/rmikehodges/SneakyVulture/sshext"
	yaml "gopkg.in/yaml.v2"
)

//Parsing Helpers//
func ParseConfig(configFile string) Config {
	var config Config
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

///////////////////

//String Slice Helper Functions//
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

func removeDuplicateStrings(inSlice []string) (outSlice []string) {
	outSlice = inSlice[:1]
	for _, p := range inSlice {
		inOutSlice := false
		for _, q := range outSlice {
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

func splitOnComma(inString string) (outSlice []string) {
	outSlice = strings.Split(inString, ",")
	return
}

//////////////////////////////////

//Structs in use throughout the application//
type Config struct {
	PublicKey   string `yaml:"PublicKey"`
	Customer    string `yaml:"Customer"`
	StartPort   int    `yaml:"StartPort"`
	Proxychains string `yaml:"Proxychains"`
	CSDir       string `yaml:"CobaltStrikeDir"`
	CSProfiles  string `yaml:"CobaltStrikeProfiles"`
	CSLicense   string `yaml:"CobaltStrikeLicense"`
	NmapDir     string `yaml:"NmapDir"`
	AWS         struct {
		Secret         string `yaml:"secret"`
		AccessID       string `yaml:"accessID"`
		Regions        string `yaml:"regions"`
		ImageIDs       string `yaml:"imageIDs"`
		Type           string `yaml:"type"`
		Number         int    `yaml:"number"`
		Termination    map[string][]string
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

type Instance struct {
	Cloud struct {
		Config      Config
		Type        string
		Description string
		ID          string
		Region      string
		IPv4        string
		Firewalls   []string
		Domain      string
	}
	SSH struct {
		Username   string
		PrivateKey string
	}
	Proxy struct {
		SOCKSActive bool
		SOCKSPort   string
	}
	SSL struct {
		SSLKeyPass   string
		CertLocation string
		SSLEnabled   bool
	}
	Nmap struct {
		NmapTargets map[string][]string
		NmapActive  bool
		NmapCmd     string
		NmapProcess string
		TimeWindow  string
	}
	System struct {
		CMD     *exec.Cmd
		Stderr  *bufio.Reader
		HomeDir string
		NmapDir string
	}
	DomainFront struct {
		DomainFrontURL string
		DomainFront    string
	}
	CobaltStrike struct {
		C2Profile         string
		CSPassword        string
		TeamserverEnabled bool
		KillDate          string
	}
}

////////////////////////////////////////

//AWS Specific Helpers//
func (config *Config) updateTermination(terminationMap map[string][]string) {
	config.AWS.Termination = terminationMap
}

func ec2ToInstance(runResult []*ec2.Instance, regionMap map[string][]string) (ec2Instances []Instance) {
	var ec2Instance Instance
	for region, instanceIDs := range regionMap {
		for _, instance := range runResult {
			if contains(instanceIDs, *instance.InstanceId) {
				ec2Instance.Cloud.ID = aws.StringValue(instance.InstanceId)
				ec2Instance.Cloud.IPv4 = aws.StringValue(instance.PublicIpAddress)
				ec2Instance.Cloud.Type = "EC2"
				ec2Instance.SSH.Username = "ubuntu"
				ec2Instance.Cloud.Region = region
				ec2Instances = append(ec2Instances, ec2Instance)
			}
		}
	}
	return
}

///////////////////////

//DO Specified Helpers//
func dropletsToInstances(droplets []godo.Droplet, config Config) []Instance {
	var Instances []Instance
	for _, drop := range droplets {

		IP, err := drop.PublicIPv4()
		if err != nil {
			log.Fatalf("Unable to get ip address for %s", drop)
		}
		tempInstance := Instance{}
		tempInstance.Cloud.Type = "DO"
		tempInstance.Cloud.ID = strconv.Itoa(drop.ID)
		tempInstance.Cloud.Region = drop.Region.Slug
		tempInstance.Cloud.IPv4 = IP

		tempInstance.SSH.Username = "root"
		Instances = append(Instances, tempInstance)
	}
	return Instances
}

//Instance Helpers//
func StopInstances(config Config, allInstances []*Instance) {
	for _, instance := range allInstances {
		if instance.Cloud.Type == "DO" {
			id, _ := strconv.Atoi(instance.Cloud.ID)
			do.DestroyDOInstance(config.DO.Token, id)
		}
	}
	fmt.Println("About to terminate")
	amazon.TerminateEC2Instances(config.AWS.Termination, config.AWS.Secret, config.AWS.AccessID)
	for _, instance := range allInstances {
		if instance.Proxy.SOCKSActive == true && instance.System.CMD.Process != nil {
			error := instance.System.CMD.Process.Kill()
			instance.Proxy.SOCKSActive = false
			if error != nil {
				fmt.Println("Error killing socks process")
				fmt.Println(error)
			}
		}
	}
}

//Probaly Unecessary
// func combineToMap(allInstances []Instance) map[int]*Instance {
// 	instanceMap := make(map[int]*Instance)
// 	for i := range allInstances {
// 		instanceMap[i] = &allInstances[i]
// 	}
// 	return instanceMap
// }

func getIPAddresses(allInstances []*Instance, config Config) {
	for _, instance := range allInstances {
		if instance.Cloud.Type == "EC2" {
			instance.Cloud.IPv4 = amazon.GetEC2IP(instance.Cloud.Region, config.AWS.Secret, config.AWS.AccessID, instance.Cloud.ID)
		}
		if instance.Cloud.Type == "DO" {
			doID, _ := strconv.Atoi(instance.Cloud.ID)
			instance.Cloud.IPv4 = do.GetDOIP(config.DO.Token, doID)
		}
	}
	// return allInstances
}

func StartInstances(config Config) ([]*Instance, map[string][]string) {
	var cloudInstances []Instance
	var instanceArray []*Instance
	var terminationMap map[string][]string
	var ec2Instances []*ec2.Instance
	if config.AWS.Number > 0 {
		ec2Instances, terminationMap = amazon.DeployMultipleEC2(config.AWS.Secret, config.AWS.AccessID, splitOnComma(config.AWS.Regions), splitOnComma(config.AWS.ImageIDs), config.AWS.Number, config.PublicKey, config.AWS.Type)
		cloudInstances = append(cloudInstances, ec2ToInstance(ec2Instances, terminationMap)...)
	}
	if config.DO.Number > 0 {
		doInstances := do.DeployDO(config.DO.Token, config.DO.Regions, config.DO.Memory, config.DO.Slug, config.DO.Fingerprint, config.DO.Number, config.DO.Name)
		cloudInstances = append(cloudInstances, dropletsToInstances(doInstances, config)...)
	}
	if len(cloudInstances) > 0 {
		fmt.Println("Waiting a few seconds for all instances to initialize...")
		time.Sleep(60 * time.Second)
		for i := range cloudInstances {
			instanceArray = append(instanceArray, &cloudInstances[i])
		}
		getIPAddresses(instanceArray, config)
	}
	return instanceArray, terminationMap
}

func Initialize(allInstances []*Instance, config Config) {
	for _, instance := range allInstances {
		instance.SSH.PrivateKey = strings.Split(config.PublicKey, ".pub")[0]
		instance.System.HomeDir = sshext.SetHomeDir(instance.Cloud.IPv4, instance.SSH.Username, instance.SSH.PrivateKey)
		instance.Proxy.SOCKSActive = false
		instance.CobaltStrike.TeamserverEnabled = false
		instance.Nmap.NmapActive = false
	}
}

//Proxies//
func CreateSOCKS(Instances []*Instance, startPort int) (string, string) {
	socksConf := make(map[int]string)
	counter := startPort
	for _, instance := range Instances {
		instance.Proxy.SOCKSActive = sshext.CreateSingleSOCKS(instance.SSH.PrivateKey, instance.SSH.Username, instance.Cloud.IPv4, counter)
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
				instance.System.HomeDir, localDir, strings.Join(ports, "-"), additionalOpts,
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
