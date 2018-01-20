package helper

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
	"github.com/rmikehodges/SneakyVulture/amazon"
	yaml "gopkg.in/yaml.v2"
)

//Parsing Helpers//
func parseConfig() Config {
	var config Config
	data, err := ioutil.ReadFile("config/config.yaml")
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
	fmt.Println(inSlice)
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

func ec2ToInstance([]ec2.Instance) {
	var ec2Instances []cloud.Instance
	var ec2Instance cloud.Instance
	for _, instance := range runResult.Instances {
		ec2Instance.Cloud.ID = aws.StringValue(instance.InstanceId)
		ec2Instance.Cloud.IPv4 = aws.StringValue(instance.PublicIpAddress)
		ec2Instance.Cloud.Type = "EC2"
		ec2Instance.SSH.Username = "ubuntu"
		ec2Instance.Proxy.SOCKSActive = false
		ec2Instance.SOCKSPort = "0"
		ec2Instance.Cloud.Region = region
		ec2Instance.SSH.PrivateKey = strings.Join(privKey[:len(privKey)-1], ".")
		ec2Instance.Cloud.IPv4 = config
		ec2Instances = append(ec2Instances, ec2Instance)
	}
}

///////////////////////

//Instance Helpers//
func stopInstances(config Config, allInstances map[int]*Instance) map[int]*Instance {
	for _, instance := range allInstances {
		if instance.Cloud.Type == "DO" {
			destroyDOInstance(*instance)
		}
	}
	fmt.Println("About to terminate")
	amazon.TerminateEC2Instances(config.AWS.Termination, config.AWS.Secret, config.AWS.AccessID)
	for p := range allInstances {
		if allInstances[p].Proxy.SOCKSActive == true && allInstances[p].System.CMD.Process != nil {
			error := allInstances[p].CMD.Process.Kill()
			allInstances[p].Proxy.SOCKSActive = false
			if error != nil {
				fmt.Println("Error killing socks process")
				fmt.Println(error)
			}
		}
	}
	return allInstances
}

func combineToMap(allInstances []Instance) map[int]*Instance {
	instanceMap := make(map[int]*Instance)
	for i := range allInstances {
		instanceMap[i] = &allInstances[i]
	}
	return instanceMap
}

func getIPAddresses(allInstances map[int]*Instance, config Config) {
	for k := range allInstances {
		if allInstances[k].Cloud.Type == "EC2" {
			allInstances[k].Cloud.IPv4 = getEC2IP(allInstances[k].Cloud.Region, config.AWS.Secret, config.AWS.AccessID, allInstances[k].Cloud.ID)
		}
		if allInstances[k].Cloud.Type == "DO" {
			doID, _ := strconv.Atoi(allInstances[k].Cloud.ID)
			allInstances[k].Cloud.IPv4 = getDOIP(config.DO.Token, doID)
		}
	}
	// return allInstances
}
func startInstances(config Config) (map[int]*Instance, map[string][]string) {
	ec2Result := 0
	doResult := 0
	var terminationMap map[string][]string
	var ec2Instances []Instance
	var doInstances []Instance
	var allInstances []Instance
	var mappedInstances map[int]*Instance
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
