package google

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// --machine-type=MACHINE_TYPE
// Specifies the machine type used for the instances. To get a list of available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type is n1-standard-1.

func CreateInstances(description string, name string, count int, machineType string, zones string, imageFamily string,
	project string, imageProject string, publicKey string) [][]string {

	//Ensuring the ssh key is set for the project
	setSSHKeyFile(strings.Split(publicKey, ".pub")[0], project)

	zoneMap := zoneMap(strings.Split(zones, ","), count)
	nameList := ""
	var instances [][]string
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	//For each zone create the correct amount of instances
	for zone, counter := range zoneMap {

		//Creating the naming for the CLI
		for i := 0; i < counter; i++ {
			nameList = nameList + name + strconv.Itoa(r1.Intn(1000000)) + " "
		}
		nameList := nameList[:len(nameList)-1]

		//Command
		gcloudCreate := `gcloud compute instances create ` + nameList + ` --machine-type=` + machineType + ` --description=` + description + ` --zone=` + zone + ` --image-project=` + imageProject + ` --project=` + project + ` | grep -v https://www.googleapis.com | grep -v INTERNAL_IP`
		cmd := exec.Command("bash", "-c", gcloudCreate)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error creating an instance for zone "+zone+": %s", err)
		}

		//Splitting the output based on newlines
		splitResult := strings.Split(string(output), "\n")
		splitResult = splitResult[:len(splitResult)-1]
		var temp []string

		//Looping through split output and splitting it based on " "
		for _, p := range splitResult {
			parsed := strings.Split(p, " ")

			for _, q := range parsed {
				if strings.TrimSpace(q) != "" {
					temp = append(temp, q)
				}
			}
			instances = append(instances, temp)
			temp = []string{}
		}

	}

	return instances
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

func stopInstances(project string, instanceNames []string, zone string) bool {
	//
	instances := strings.Join(instanceNames, " ")
	cmd := exec.Command("gcloud", "compute", "instances", "stop", instances, "--project", project, "--zone", zone)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error stopping instances: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

//stopInstancesMultipleZones takes a mapping of instance names to their respective zone
//and passes them to stop instance to stop it
func stopInstancesMultipleZones(project string, zoneInstanceMap map[string][]string) {
	for zone, instances := range zoneInstanceMap {
		stopInstances(project, instances, zone)
	}
}

func startInstancesMultipleZones(description string, machineType string, zone string, imageFamily string,
	project string, imageProject string) {
}

//DestroyInstance destroys the given instances for the specified zones
func DestroyInstance(project string, instanceNames []string, zone string) error {
	instances := strings.Join(instanceNames, " ")
	gcloudDelete := "echo 'Y' | gcloud compute instances delete " + instances + " --project " + project + " --zone " + zone
	fmt.Println(gcloudDelete)
	cmd := exec.Command("bash", "-c", gcloudDelete)
	err := cmd.Start()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err != nil {
		fmt.Printf("Error stopping instances: %s", err)
		return err
	}
	return nil
}

func DestroyMultipleInstances(project string, zoneInstanceMap map[string][]string) {
	for zone, instances := range zoneInstanceMap {
		stopInstances(project, instances, zone)
	}
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

//As of right now firewall rules apply globally
//to all compute cloud instances as they are all on the
//same network. This probably will be changed in the future
func createInboundRule(project string, ruleName string, protocol string, port string, sourceRanges string) bool {
	parsedSourceRanges := sourceRanges
	if !verifySourceRanges(sourceRanges) {
		parsedSourceRanges = "0.0.0.0/0"
	}
	rule := protocol + ":" + port
	cmd := exec.Command("gcloud", "compute", "firewall-rules", "create", ruleName, "--source-ranges", parsedSourceRanges,
		"--allow", rule)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error create firewall rule: %s", err)
		return false
	}
	//Log successful import of ssh key
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

func setSSHKeyFile(sshKeyFile string, project string) bool {
	cmd := exec.Command("gcloud", "compute", "config-ssh", "--ssh-key-file", sshKeyFile, "--project", project)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error importing sshKeyFile: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

//This needs to be done later
func checkForFirstRun() {
	//Find ~/.ssh/google_compute_engine

	//If exists then allow user to prompt through
}
