package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/jmcvetta/randutil"
	"golang.org/x/oauth2"
)

// Token implements interface for oauth2.
type Token struct {
	AccessToken string
}

////Token Functions////

// Token implements interface for oauth2.
func (t *Token) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func importDOKey(pubkey string, clinet *godo.Client) string {
	var fingerprint string

	return fingerprint
}

///////////////////////

////Machine Functions////

//Converts droplets to a Machine struct
func dropletsToCloudInstances(droplets []godo.Droplet, config Config) []CloudInstance {
	var cloudInstances []CloudInstance
	for _, drop := range droplets {

		IP, err := drop.PublicIPv4()
		if err != nil {
			log.Fatalf("Unable to get ip address for %s", drop)
		}
		privKey := strings.Split(config.PublicKey, ".")
		cloudInstances = append(cloudInstances, CloudInstance{
			Type:        "DO",
			ID:          strconv.Itoa(drop.ID),
			Region:      drop.Region.Slug,
			Username:    "root",
			IPv4:        IP,
			SOCKSActive: false,
			SOCKSPort:   "0",
			PrivateKey:  strings.Join(privKey[:len(privKey)-1], "."),
			Config:		config,
		})
	}
	return cloudInstances
}

/////////////////////////////

func getDOIP(token string, id int) string {
	client := newDOClient(token)
	machineID := id
	droplet, _, err := client.Droplets.Get(context.TODO(), machineID)
	if err != nil {
		fmt.Println("Error retrieving droplet")
	}
	IP, err := droplet.PublicIPv4()
	if err != nil {
		fmt.Println("Error retrieving droplet's IP address")
	}
	return IP
}

func destroyDOInstance(instance CloudInstance) {
	client := newDOClient(instance.Config.DO.Token)
	machineID, err := strconv.Atoi(instance.ID)
	if err != nil {
		log.Fatalf("Houston we have a problem %s", err)
	}
	_, err = client.Droplets.Delete(context.TODO(), machineID)
	if err != nil {
		log.Println("There was an error destroying the following machine, you may need to do cleanup:\n%s", instance)
	}
}

//Helper method for now
func destroyMultipleDroplets(config Config, droplets []godo.Droplet) {
	client := newDOClient(config.DO.Token)
	for _,drop := range droplets {
		client.Droplets.Delete(context.TODO(), drop.ID)
	}
	fmt.Println("Deleted all your drops")
}

//Creates New DoClient
func newDOClient(token string) *godo.Client {
	t := &Token{AccessToken: token}
	oa := oauth2.NewClient(oauth2.NoContext, t)
	return godo.NewClient(oa)
}

//Retrieves a list of available DO regions
func doRegions(client *godo.Client) ([]string, error) {
	var slugs []string
	regions, _, err := client.Regions.List(context.TODO(), &godo.ListOptions{})
	if err != nil {
		return slugs, err
	}
	for _, r := range regions {
		slugs = append(slugs, r.Slug)
	}
	return slugs, nil
}

func regionMap(slugs []string, regions string, count int) (map[string]int, error) {
	allowedSlugs := strings.Split(regions, ",")
	regionCountMap := make(map[string]int)

	if regions != "*" {
		for _, s := range slugs {
			for _, a := range allowedSlugs {
				if s == a {
					if len(regionCountMap) == count {
						break
					}
					regionCountMap[s] = 0
				}
			}
		}
	} else {
		for _, s := range slugs {
			if len(regionCountMap) == count {
				break
			}
			regionCountMap[s] = 0
		}
	}

	if len(regionCountMap) == 0 {
		return regionCountMap, errors.New("There are no regions to use")
	}

	perRegionCount := count / len(regionCountMap)
	perRegionCountRemainder := count % len(regionCountMap)

	for k := range regionCountMap {
		regionCountMap[k] = perRegionCount
	}

	if perRegionCountRemainder != 0 {
		c := 0
		for k, v := range regionCountMap {
			if c >= perRegionCountRemainder {
				break
			}
			regionCountMap[k] = v + 1
			c++
		}
	}
	return regionCountMap, nil
}

func newDropLetMultiCreateRequest(region string, config Config, count int) *godo.DropletMultiCreateRequest {
	var names []string
	for i := 0; i < count; i++ {
		name, _ := randutil.AlphaString(8)
		names = append(names, fmt.Sprintf("%s-%s", config.DO.Name, name))
	}
	return &godo.DropletMultiCreateRequest{
		Names:  names,
		Region: region,
		Size:   config.DO.Memory,
		Image: godo.DropletCreateImage{
			Slug: config.DO.Slug,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: config.DO.Fingerprint,
			},
		},
		Backups:           false,
		IPv6:              false,
		PrivateNetworking: false,
	}
}

func deployDO(config Config) ([]CloudInstance, int) {
	var droplets []godo.Droplet
	errorResult := 0
	client := newDOClient(config.DO.Token)
	availableRegions, err := doRegions(client)
	if err != nil {
		log.Fatalf("There was an error getting a list of regions:\nError: %s\n", err.Error())
	}

	regionCountMap, err := regionMap(availableRegions, config.DO.Regions, config.DO.Number)
	if err != nil {
		log.Fatalf("%s\n", err.Error())
	}
	for region, c := range regionCountMap {
		log.Printf("Creating %d droplets to region %s", c, region)
		drops, _, err := client.Droplets.CreateMultiple(context.TODO(), newDropLetMultiCreateRequest(region, config, c))
		if err != nil {
			log.Printf("There was an error creating the droplets:\nError: %s\n", err.Error())
			errorResult = 1
		}
		droplets = append(droplets, drops...)
	}
	cloudInstances := dropletsToCloudInstances(droplets, config)
	return cloudInstances, errorResult
}

//List existing droplets
func listDroplets(config Config) []godo.Droplet{
	client := newDOClient(config.DO.Token)
	droplets, _, err := client.Droplets.List(context.TODO(), &godo.ListOptions{
		Page: 1,
		PerPage: 50,
	}) 
	if err != nil {
		log.Print("There was an error retrieving droplets.")
	}
	return droplets
}



//////////////////////
//Firewalls//
/////////////////////


//Create Firewall
//TODO: Allow option for UDP port speicification
func createDOFirewall(config Config, firewallName string, ipPortMap map[string][]int) (string) {
	client := newDOClient(config.DO.Token)
	firewall := client.Firewalls

	var inbboundRules []godo.InboundRule
	var sources godo.Sources
	for ip, ports := range ipPortMap {
		sources = godo.Sources{
			Addresses: []string{ip},
		}
		for port := range ports {
			portString := strconv.Itoa(port)
			inbboundRules = append(inbboundRules, godo.InboundRule{
				Protocol: "tcp",
				PortRange: portString,
				Sources: &sources,
			})
		}
	}
	newFirewall, _, err := firewall.Create(context.TODO(), &godo.FirewallRequest{
		Name:	firewallName,
		InboundRules: inbboundRules,		
	})
	if err != nil {
		log.Println("Error encountered creating DO Firewall")
	}
	return newFirewall.ID
}


//Delete Firewall 
func deleteDOFirewall(allInstances map[int]*CloudInstance, config Config, fID string) {
	client := newDOClient(config.DO.Token)
	_, err := client.Firewalls.Delete(context.TODO(), fID)
	if err != nil {
		log.Println("Error encountered deleting: " + fID)
	}
	for i, instance := range allInstances {
		if instance.Type == "DO" && contains(instance.Firewalls, fID) {
			allInstances[i].Firewalls = removeString(allInstances[i].Firewalls, fID)
		}
	}
}

func listAllFirewalls(config Config) []godo.Firewall{
	client := newDOClient(config.DO.Token)
	firewallList, _, err := client.Firewalls.List(context.TODO(), &godo.ListOptions{
		Page: 1,
		PerPage: 1000,
	}) 
	if err != nil {
		log.Println("Error retrieving the list of DO firewalls")
	}
	return firewallList
}

func (instance *CloudInstance) listFirewallByDroplet() []godo.Firewall{
	client := newDOClient(instance.Config.DO.Token)
	instanceID, _ := strconv.Atoi(instance.ID)
	firewallList, _, err := client.Firewalls.ListByDroplet(context.TODO(), instanceID, &godo.ListOptions{
		Page: 1,
		PerPage: 1000,
	}) 
	if err != nil {
		log.Println("Error retrieving the list of DO firewalls")
	}
	return firewallList
}

//Edit Firewall
//TODO: Add Ability to add/remove firewall rules


//Change firewall belonging to instance
func (instance *CloudInstance) setDOFirewall(fwID string) {
	client := newDOClient(instance.Config.DO.Token)
	dropletID, _ := strconv.Atoi(instance.ID)
	_, err := client.Firewalls.AddDroplets(context.TODO(), fwID, dropletID)
	if err != nil {
		log.Println("Error setting the DO firewall to instance")
	}
}