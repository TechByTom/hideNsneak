package cloud

//Structs in use throughout the application//
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/digitalocean/godo"
	"github.com/rmikehodges/hideNsneak/amazon"
	"github.com/rmikehodges/hideNsneak/do"
	"github.com/rmikehodges/hideNsneak/drone"
	"github.com/rmikehodges/hideNsneak/google"
	"github.com/rmikehodges/hideNsneak/misc"
	"github.com/rmikehodges/hideNsneak/nmap"
	"github.com/rmikehodges/hideNsneak/sshext"
	yaml "gopkg.in/yaml.v2"
)

//Notes:
//Possibly add Port struct that contains the protocol i.e tcp/udp

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
		Username       string
	} `yaml:"AWS"`
	DO struct {
		Token       string `yaml:"token"`
		Fingerprint string `yaml:"fingerprint"`
		Regions     string `yaml:"regions"`
		Slug        string `yaml:"slug"`
		Memory      string `yaml:"memory"`
		Name        string `yaml:"name"`
		Number      int    `yaml:"number"`
		Username    string
	} `yaml:"DO"`
	Google struct {
		ImageURL   string `yaml:"imageURL"`
		Zones      string `yaml:"zones"`
		Number     int    `yaml:"number"`
		Project    string `yaml:"project"`
		ProjectDir string `yaml:"projectDir"`
		ClientID   string `yaml:"clientID"`
		Secret     string `yaml:"secret"`
		Username   string
	} `yaml:"Google"`
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
		State       string
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

//RegionalFirewall is a struct
//WHAT DOES THIS DO?
type RegionalFirewall struct {
	RegionPortMap map[string](map[string][]int)
}

type Firewall struct {
	FirewallType map[string]RegionalFirewall
}

type DomainFront struct {
	Type               string
	Host               string
	Target             string
	ID                 string
	ETag               string
	DistributionConfig *cloudfront.DistributionConfig
}

func ParseConfig(configFile string) *Config {
	var config *Config
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

//String() prints generic information for the user
func (instance Instance) String() string {
	socksPort := ""
	nmapActive := "N"
	if instance.Proxy.SOCKSActive {
		socksPort = instance.Proxy.SOCKSPort
	}
	if instance.Nmap.NmapActive {
		nmapActive = "Y"
	}
	returnString := fmt.Sprintf("Type: %s | IP: %s | Region: %s | Nmap Active: %s | SOCKS: %s | State: %s", instance.Cloud.Type, instance.Cloud.IPv4,
		instance.Cloud.Region, nmapActive, socksPort, instance.Cloud.State)
	return returnString
}

//Start, Stop, Initialize
func DeployInstances(config *Config, providerMap map[string]int) []*Instance {
	var cloudInstances []Instance
	var instanceArray []*Instance
	var ec2Instances []*ec2.Instance

	for provider, count := range providerMap {
		//Instance Creation
		//TODO: Add descriptions
		switch provider {
		//TODO: Catch errors on creation here
		case "AWS":
			ec2Instances = amazon.DeployInstances(config.AWS.Secret, config.AWS.AccessID,
				misc.SplitOnComma(config.AWS.Regions), misc.SplitOnComma(config.AWS.ImageIDs), count,
				config.PublicKey, config.AWS.Type)
			cloudInstances = append(cloudInstances, ec2ToInstance(ec2Instances, config.AWS.Username)...)
		case "DO":
			doInstances, _ := do.DeployInstances(config.DO.Token, config.DO.Regions, config.DO.Memory, config.DO.Slug,
				config.DO.Fingerprint, count, config.DO.Name)
			cloudInstances = append(cloudInstances, dropletsToInstances(doInstances, config)...)
		case "Azure":
			//To be added
		case "Google":
			googleInstances := google.DeployInstances("", config.Customer, count, config.Google.Zones, config.Google.ImageURL,
				config.Google.Project, config.PublicKey, config.Google.ClientID, config.Google.Secret)
			cloudInstances = append(cloudInstances, googleToInstance(googleInstances, config.Google.Username)...)
		default:
			continue
		}
	}
	if len(cloudInstances) > 0 {
		fmt.Println("Waiting a few seconds for all instances to initialize...")
		time.Sleep(60 * time.Second)
		for i := range cloudInstances {
			instanceArray = append(instanceArray, &cloudInstances[i])
		}

		getIPAddresses(instanceArray, config)

		//Logging Creation of instances
		for _, instance := range instanceArray {
			instance.Cloud.State = "RUNNING"
			misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " created")
		}
	}

	Initialize(instanceArray, config)

	return instanceArray
}

//TODO: Add Destruction of Azure instances
func DestroyInstances(config *Config, allInstances []*Instance) {
	for _, instance := range allInstances {
		switch instance.Cloud.Type {
		case "DO":
			id, _ := strconv.Atoi(instance.Cloud.ID)
			if err := do.DestroyInstance(config.DO.Token, id); err != nil {
				fmt.Println(instance.String() + " not destroyed properly")
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
				misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
			} else {
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
			}
		case "AWS":
			if err := amazon.DestroyInstance(instance.Cloud.Region, instance.Cloud.ID, config.AWS.Secret, config.AWS.AccessID); err != nil {
				fmt.Println(instance.String() + " not destroyed properly")
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
				misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
			} else {
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
			}
		case "Google":
			if err := google.DestroyInstance(instance.Cloud.ID, instance.Cloud.Region, config.Google.Project,
				config.Google.ClientID, config.Google.Secret); err != nil {
				fmt.Println(instance.String() + " not destroyed properly")
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
				misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
			} else {
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
			}
		case "Azure":
			//TODOL Implement destruction of Azure
			fmt.Println("Implement Azure")
		default:
			fmt.Println("Unknown Provider...skipping..")
		}
		if instance.Proxy.SOCKSActive {
			if stopSingleSOCKS(instance) == false {
				fmt.Println("Error: SOCKS Proxy not killed for " + instance.Cloud.IPv4 + " check application logs")
			}
		}
	}
}

func StartInstances(config *Config, allInstances []*Instance) {
	for _, instance := range allInstances {
		if instance.Cloud.State == "STOPPED" {
			switch instance.Cloud.Type {
			case "DO":
				id, _ := strconv.Atoi(instance.Cloud.ID)
				if err := do.StartInstance(config.DO.Token, id); err != nil {
					fmt.Println(instance.String() + " not started properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not started - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "RUNNING"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " started")
				}
			case "AWS":
				if err := amazon.StartInstance(instance.Cloud.Region, instance.Cloud.ID, config.AWS.Secret, config.AWS.AccessID); err != nil {
					fmt.Println(instance.String() + " not started properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not started - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "RUNNING"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " started")
				}
			case "Google":
				if err := google.StartInstance(instance.Cloud.ID, instance.Cloud.Region, config.Google.Project,
					config.Google.ClientID, config.Google.Secret); err != nil {
					fmt.Println(instance.String() + " not destroyed properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "RUNNING"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
				}
			case "Azure":
				//TODOL Implement destruction of Azure
				fmt.Println("Implement Azure")
			default:
				fmt.Println("Unknown Provider...skipping..")
			}
		}
	}

}

func StopInstances(config *Config, allInstances []*Instance) {
	for _, instance := range allInstances {
		if instance.Cloud.State == "RUNNING" {
			switch instance.Cloud.Type {
			case "DO":
				id, _ := strconv.Atoi(instance.Cloud.ID)
				if err := do.StopInstance(config.DO.Token, id); err != nil {
					fmt.Println(instance.String() + " not stopped properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not stopped - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "STOPPED"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " stopped")
				}
			case "AWS":
				if err := amazon.StopInstance(instance.Cloud.Region, instance.Cloud.ID, config.AWS.Secret, config.AWS.AccessID); err != nil {
					fmt.Println(instance.String() + " not stopped properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not stopped - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "STOPPED"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " stopped")
				}
			case "Google":
				if err := google.StopInstance(instance.Cloud.ID, instance.Cloud.Region, config.Google.Project,
					config.Google.ClientID, config.Google.Secret); err != nil {
					fmt.Println(instance.String() + " not stopped properly")
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not stopped - see error log")
					misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + fmt.Sprint(err))
				} else {
					instance.Cloud.State = "STOPPED"
					misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " stopped")
				}
			case "Azure":
				//TODOL Implement destruction of Azure
				fmt.Println("Implement Azure")
			default:
				fmt.Println("Unknown Provider...skipping..")
			}
			if instance.Proxy.SOCKSActive {
				if stopSingleSOCKS(instance) == false {
					fmt.Println("Error: SOCKS Proxy not killed for " + instance.Cloud.IPv4 + " check application logs")
				}
			}
		}
	}

}

//stopSocks loops through a set of instances and kills their SOCKS processes
func stopAllSOCKS(allInstances []*Instance) {
	for _, instance := range allInstances {
		if instance.Proxy.SOCKSActive == true && instance.Proxy.Process != nil {
			err := instance.Proxy.Process.Kill()
			if err != nil {
				misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " SOCKS not destroyed- see error log")
				misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + "Error killing SOCKS process:" + fmt.Sprint(err))
				continue
			}
			instance.Proxy.SOCKSActive = false
			misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " : SOCKS destroyed")
		}
	}
}

func stopSingleSOCKS(instance *Instance) bool {
	err := instance.Proxy.Process.Kill()
	if err != nil {
		misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " SOCKS not destroyed- see error log")
		misc.WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + "Error killing SOCKS process:" + fmt.Sprint(err))
		return false
	}
	instance.Proxy.SOCKSActive = false
	misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " : SOCKS destroyed")
	return true
}

func Initialize(allInstances []*Instance, config *Config) {
	for _, instance := range allInstances {
		instance.SSH.PrivateKey = strings.Split(config.PublicKey, ".pub")[0]

		// instance.Cloud.HomeDir = sshext.SetHomeDir(instance.Cloud.IPv4, instance.SSH.Username, instance.SSH.PrivateKey)
		instance.Proxy.SOCKSActive = false
		instance.CobaltStrike.TeamserverEnabled = false
		instance.Nmap.NmapActive = false
		misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " initialized")
		instance.Cloud.State = "RUNNING"
	}
}

//Converting Custom cloud objects to Instance objects
func dropletsToInstances(droplets []godo.Droplet, config *Config) []Instance {
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

		tempInstance.SSH.Username = config.DO.Username
		Instances = append(Instances, tempInstance)
	}
	return Instances
}

func ec2ToInstance(runResult []*ec2.Instance, username string) (ec2Instances []Instance) {
	var ec2Instance Instance
	for _, instance := range runResult {
		availZone := aws.StringValue(instance.Placement.AvailabilityZone)
		ec2Instance.Cloud.ID = aws.StringValue(instance.InstanceId)
		ec2Instance.Cloud.IPv4 = aws.StringValue(instance.PublicIpAddress)
		ec2Instance.Cloud.Type = "AWS"
		ec2Instance.SSH.Username = username
		ec2Instance.Cloud.Region = availZone[:len(availZone)-1]
		ec2Instances = append(ec2Instances, ec2Instance)
	}
	return
}

func googleToInstance(googleInstances []*google.GoogleInstance, username string) (instances []Instance) {
	var instance Instance
	for _, googleInstance := range googleInstances {
		instance.Cloud.ID = googleInstance.ID
		instance.Cloud.IPv4 = googleInstance.IPv4
		instance.Cloud.Type = "Google"
		instance.SSH.Username = username
		instance.Cloud.Region = googleInstance.Zone
		instance.Cloud.State = googleInstance.State
		instances = append(instances, instance)
	}
	return instances
}

func getIPAddresses(allInstances []*Instance, config *Config) {
	for _, instance := range allInstances {
		if instance.Cloud.Type == "EC2" {
			instance.Cloud.IPv4 = amazon.GetEC2IP(instance.Cloud.Region, config.AWS.Secret, config.AWS.AccessID, instance.Cloud.ID)
		}
		if instance.Cloud.Type == "DO" {
			doID, _ := strconv.Atoi(instance.Cloud.ID)
			instance.Cloud.IPv4 = do.GetDOIP(config.DO.Token, doID)
		}
		if instance.Cloud.Type == "Google" {
			instance.Cloud.IPv4 = google.GetIPAddress(instance.Cloud.Region, instance.Cloud.ID, config.Google.Secret,
				config.Google.ClientID, config.Google.Project)
		}
	}
}

//RUNNERS//
func CreateSOCKS(Instances []*Instance, startPort int) (string, string) {
	socksConf := make(map[int]string)
	counter := startPort
	for _, instance := range Instances {
		instance.Proxy.SOCKSActive, instance.Proxy.Process = sshext.CreateSingleSOCKS(instance.SSH.PrivateKey, instance.SSH.Username, instance.Cloud.IPv4, counter)
		if instance.Proxy.SOCKSActive {
			misc.WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " SOCKS Created")
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
func RunConnectScans(instances []*Instance, output string, additionalOpts string, evasive bool, scope string,
	ports []string, localDir string, droneImport bool) {
	targets := nmap.ParseIPFile(scope)
	ipPorts := nmap.GenerateIPPortList(targets, ports)
	if evasive == true {
		fmt.Println("Evasive")
		nmapTargeting := nmap.RandomizeIPPortsToHosts(len(instances), ipPorts)
		for i, instance := range instances {
			go nmap.InitiateConnectScan(instance.SSH.Username, instance.Cloud.IPv4, instance.SSH.PrivateKey, nmapTargeting[i],
				instance.Cloud.HomeDir, localDir, additionalOpts, evasive, instance.Cloud.Type, instance.Cloud.Region)

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

func ImportNmaps(localDir string, insecureSSL bool, limitHosts bool, forcePorts bool, lairPID string, tags string) {
	importResult := false
	xmlFiles := nmap.ListNmapXML(localDir)
	for _, xmlFile := range xmlFiles {
		for !importResult {
			importResult = drone.NmapImport(insecureSSL, limitHosts, forcePorts, xmlFile, lairPID, tags)
		}
	}
}

func CreateFirewall(instances []*Instance, config Config, ports []int, groupName string, desc string) {
	firewall := RegionalFirewall{}
	for _, instance := range instances {
		switch instance.Cloud.Type {
		case "AWS":
			firewall.RegionPortMap["AWS"][instance.Cloud.IPv4] = ports
		case "DO":
			firewall.RegionPortMap["DO"][instance.Cloud.IPv4] = ports
		case "Google":
			firewall.RegionPortMap["Google"][instance.Cloud.IPv4] = ports
		case "Azure":
			firewall.RegionPortMap["Azure"][instance.Cloud.IPv4] = ports
		default:
			fmt.Println("Unknown instance type, skpping")
		}
	}
	//Don't know what this is used for
	// for _, firewallType := range firewall.RegionPortMap {
	// 	switch firewallType {
	// 	case "AWS":
	// 	}
	// }

}

//CreateCloudfront is a runner function for the creation of amazon cloudfront
func CreateCloudfront(config *Config, domain string) DomainFront {
	var cloudFront DomainFront
	tempDistribution, etag, err := amazon.CreateCloudFront(config.Customer, "", domain, config.AWS.Secret, config.AWS.AccessID)
	if err != nil {
		fmt.Printf("There was a problem creating the cloudfront distribution: %s", err)
		return cloudFront
	}
	cloudFront.Type = "AWS"
	cloudFront.ETag = etag
	cloudFront.Host = *tempDistribution.DomainName
	cloudFront.DistributionConfig = tempDistribution.DistributionConfig
	return cloudFront
}

//CreateGoogleDomainFront is a wrapper for the google package
//TODO Implement logging
func CreateGoogleDomainFront(config *Config, domain string, keystore string, keystorePass string,
	newProject bool, restrictedUserAgent string, restrictedSubnet string, restrictedHeader string,
	defaultRedirect string, c2Profile string) string {
	fmt.Println(config.Google.ProjectDir)
	result, url := google.CreateRedirector(config.Google.Project, restrictedUserAgent, restrictedSubnet, restrictedHeader,
		defaultRedirect, domain, newProject, config.Google.ProjectDir, c2Profile, c2Profile+"-2",
		keystore, keystorePass)
	if result {
		return url
	}
	return ""
}
