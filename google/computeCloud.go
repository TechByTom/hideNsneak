package google

import (
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"
)

// --machine-type=MACHINE_TYPE
// Specifies the machine type used for the instances. To get a list of available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type is n1-standard-1.

type GoogleInstance struct {
	ID      string
	Zone    string
	IPv4    string
	State   string
	Project string
}

func DeployInstances(description string, name string, count int, zones string, image string,
	project string, publicKey string, clientID string, secret string) []*GoogleInstance {

	auth := Authentication{
		ClientID: clientID,
		Secret:   secret,
		Project:  project,
	}
	prefix := "https://www.googleapis.com/compute/v1/projects/" + project
	imageURL := image

	service := computeAuth(auth)

	var googleInstances []*GoogleInstance

	//TODO: Add this to config file

	//TODO: Make sure SSH key file is added to the project

	zoneMap := zoneMap(strings.Split(zones, ","), count)

	// var instances [][]string
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	//For each zone create the correct amount of instances
	for zone, counter := range zoneMap {
		//Creating the naming for the CLI
		for i := 0; i < counter; i++ {
			instance := &compute.Instance{
				Name:        name + strconv.Itoa(r1.Intn(1000000)),
				Description: "",
				MachineType: prefix + "/zones/" + zone + "/machineTypes/n1-standard-1",
				Disks: []*compute.AttachedDisk{
					{
						AutoDelete: true,
						Boot:       true,
						Type:       "PERSISTENT",
						InitializeParams: &compute.AttachedDiskInitializeParams{
							DiskName:    name + strconv.Itoa(r1.Intn(1000000)),
							SourceImage: imageURL,
						},
					},
				},
				NetworkInterfaces: []*compute.NetworkInterface{
					&compute.NetworkInterface{
						AccessConfigs: []*compute.AccessConfig{
							&compute.AccessConfig{
								Type: "ONE_TO_ONE_NAT",
								Name: "External NAT",
							},
						},
						Network: prefix + "/global/networks/default",
					},
				},
			}
			res, err := service.Instances.Insert(project, zone, instance).Do()
			if err != nil {
				//Log here
				fmt.Printf("Error creating instance: %s", err)
				continue
			}
			if res.HTTPStatusCode != 200 {
				//Log here
				continue
			}

			tempInstance, _ := service.Instances.Get(project, zone, instance.Name).Do()
			fmt.Println(tempInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP)
			googleInstance := &GoogleInstance{
				ID:      instance.Name,
				Zone:    zone,
				IPv4:    tempInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP,
				State:   tempInstance.Status,
				Project: project,
			}
			googleInstances = append(googleInstances, googleInstance)
		}
	}
	return googleInstances
}

func DestroyInstance(name string, zone string, project string, clientID string, secret string) error {
	auth := Authentication{
		ClientID: clientID,
		Secret:   secret,
		Project:  project,
	}
	service := computeAuth(auth)
	resp, err := service.Instances.Delete(project, zone, name).Do()
	if err != nil {
		return err
	}
	if resp.HTTPStatusCode != 200 {
		return fmt.Errorf("Error: Response code %d", resp.HTTPStatusCode)
	}
	return nil
}

func StartInstance(name string, zone string, project string, clientID string, secret string) error {
	auth := Authentication{
		ClientID: clientID,
		Secret:   secret,
		Project:  project,
	}
	service := computeAuth(auth)
	resp, err := service.Instances.Start(project, zone, name).Do()
	if err != nil {
		return err
	}
	if resp.HTTPStatusCode != 200 {
		return fmt.Errorf("Error: Response code %d", resp.HTTPStatusCode)
	}
	return nil
}

func StopInstance(name string, zone string, project string, clientID string, secret string) error {
	auth := Authentication{
		ClientID: clientID,
		Secret:   secret,
		Project:  project,
	}
	service := computeAuth(auth)
	resp, err := service.Instances.Stop(project, zone, name).Do()
	if err != nil {
		return err
	}
	if resp.HTTPStatusCode != 200 {
		return fmt.Errorf("Error: Response code %d", resp.HTTPStatusCode)
	}
	return nil
}

//As of right now firewall rules apply globally
//to all compute cloud instances as they are all on the
//same network. This probably will be changed in the future
func createInboundRule(project string, ruleName string, protocol string, port string, sourceRanges string) bool {
	// parsedSourceRanges := sourceRanges
	// if !verifySourceRanges(sourceRanges) {
	// 	parsedSourceRanges = "0.0.0.0/0"
	// }
	// rule := protocol + ":" + port
	return true
}

func deleteInboundRule(project string, ruleName string) bool {
	cmd := exec.Command("gcloud", "compute", "firewall-rules", "delete", ruleName)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error create firewall rule: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

func describeRules(project string) bool {
	cmd := exec.Command("gcloud", "compute", "firewall-rules", "describe")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error describe firewall rules: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

//Outbound rule placeholders just in case I decide to make them
func createOutboundRule() {}

func deleteOutboundRule() {}

func GetIPAddress(zone string, id string, secret string, clientID string, project string) string {
	auth := Authentication{
		ClientID: clientID,
		Secret:   secret,
		Project:  project,
	}
	service := computeAuth(auth)
	tempInstance, _ := service.Instances.Get(project, zone, id).Do()
	return tempInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP
}

func zoneMap(zones []string, count int) map[string]int {
	regionCountMap := make(map[string]int)

	perRegionCount := count / len(zones)
	perRegionCountRemainder := count % len(zones)

	if perRegionCount == 0 {
		regionCountMap[zones[0]] = perRegionCountRemainder
	} else {
		counter := perRegionCountRemainder
		for _, zone := range zones {
			regionCountMap[zone] = perRegionCount
			if counter != 0 {
				regionCountMap[zone] = regionCountMap[zone] + 1
				counter--
			}
		}
	}
	return regionCountMap
}

//Helper function for firewall functions to verify correct CIDR format
func verifySourceRanges(sourceRanges string) bool {
	sourceList := strings.Split(sourceRanges, ",")
	r, _ := regexp.Compile(`([0-255]{1,3}\.){3}[0-255]{1,3}\/[0-32]{1,2}`)
	for _, cidr := range sourceList {
		if !r.Match([]byte(cidr)) {
			return false
		}
	}
	return true
}
