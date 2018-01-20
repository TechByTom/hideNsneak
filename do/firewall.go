package do

import (
	"context"
	"log"
	"strconv"

	"github.com/digitalocean/godo"
)

//////////////////////
//Firewalls//
/////////////////////

//Create Firewall
//TODO: Allow option for UDP port speicification
func createDOFirewall(token string, firewallName string, ipPortMap map[string][]int) string {
	client := newDOClient(token)
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
				Protocol:  "tcp",
				PortRange: portString,
				Sources:   &sources,
			})
		}
	}
	newFirewall, _, err := firewall.Create(context.TODO(), &godo.FirewallRequest{
		Name:         firewallName,
		InboundRules: inbboundRules,
	})
	if err != nil {
		log.Println("Error encountered creating DO Firewall")
	}
	return newFirewall.ID
}

//Delete Firewall
func deleteDOFirewall(token string, fID string) bool {
	client := newDOClient(token)
	_, err := client.Firewalls.Delete(context.TODO(), fID)
	if err != nil {
		log.Println("Error encountered deleting: " + fID)
		return false
	}
	return true
}

func listAllFirewalls(token string) []godo.Firewall {
	client := newDOClient(token)
	firewallList, _, err := client.Firewalls.List(context.TODO(), &godo.ListOptions{
		Page:    1,
		PerPage: 1000,
	})
	if err != nil {
		log.Println("Error retrieving the list of DO firewalls")
	}
	return firewallList
}

func listFirewallByDroplet(token string, id int) []godo.Firewall {
	client := newDOClient(token)
	firewallList, _, err := client.Firewalls.ListByDroplet(context.TODO(), id, &godo.ListOptions{
		Page:    1,
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
func setDOFirewall(fwID string, token string, id int) {
	client := newDOClient(token)
	_, err := client.Firewalls.AddDroplets(context.TODO(), fwID, id)
	if err != nil {
		log.Println("Error setting the DO firewall to instance")
	}
}
