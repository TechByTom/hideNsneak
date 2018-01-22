package cloud

//Structs in use throughout the application//
import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/digitalocean/godo"
	"github.com/rmikehodges/SneakyVulture/amazon"
	"github.com/rmikehodges/SneakyVulture/do"
	"github.com/rmikehodges/SneakyVulture/sshext"
)

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
		HomeDir     string
	}
	SSH struct {
		Username   string
		PrivateKey string
	}
	Proxy struct {
		SOCKSActive bool
		SOCKSPort   string
		Process     *os.Process
	}
	SSL struct {
		SSLKeyPass   string
		CertLocation string
		SSLEnabled   bool
	}
	Nmap struct {
		NmapTargets  map[string][]string
		NmapActive   bool
		NmapCmd      string
		NmapProcess  string
		TimeWindow   string
		NmapLocalDir string
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

func (config *Config) updateTermination(terminationMap map[string][]string) {
	config.AWS.Termination = terminationMap
}

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
		instance.Cloud.HomeDir = sshext.SetHomeDir(instance.Cloud.IPv4, instance.SSH.Username, instance.SSH.PrivateKey)
		instance.Proxy.SOCKSActive = false
		instance.CobaltStrike.TeamserverEnabled = false
		instance.Nmap.NmapActive = false
	}
}

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
		if instance.Proxy.SOCKSActive == true && instance.Proxy.Process != nil {
			error := instance.Proxy.Process.Kill()
			instance.Proxy.SOCKSActive = false
			if error != nil {
				fmt.Println("Error killing socks process")
				fmt.Println(error)
			}
		}
	}
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
