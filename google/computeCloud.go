package google

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// --machine-type=MACHINE_TYPE
// Specifies the machine type used for the instances. To get a list of available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type is n1-standard-1.

func createInstance(description string, machineType string, zone string, imageFamily string,
	project string, imageProject string) {
	cmd := exec.Command("gcloud", "compute", "instances", "create", "--machine-type="+machineType, "--description="+description,
		"--zone="+zone, "--image-family="+imageFamily, "--image-project="+imageProject, "--project="+project)
	//Need to read output to map it to cloudInstance object
	cmd.Stdout = os.Stdout
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
	project string, imageProject string){

	}


func destroyInstance(project string, instanceNames []string, zone string) bool {
	instances := strings.Join(instanceNames, " ")
	cmd := exec.Command("gcloud", "compute", "instances", "delete", instances, "--project", project, "--zone", zone)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error stopping instances: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

func destroyMultiple(project string, zoneInstanceMap map[string][]string) {
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

func deleteInboundRule(project string, ruleName string) {
	cmd := exec.Command("gcloud", "compute", "firewall-rules", "delete", ruleName)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error create firewall rule: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}

func describeRules(project string) {
	cmd := exec.Command("gcloud", "compute", "firewall-rules", "describe")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error describe firewall rules: %s", err)
		return false
	}
	//Log successful import of ssh key
	return true
}
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
