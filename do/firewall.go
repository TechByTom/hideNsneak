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
func CreateDOFirewall(token string, firewallName string, ips []string, ports []int) (string, error) {
	client := newDOClient(token)
	firewall := client.Firewalls

	var inbboundRules []godo.InboundRule
	var sources godo.Sources
	sources = godo.Sources{
		Addresses: ips,
	}
	for _, port := range ports {

		portString := strconv.Itoa(port)
		inbboundRules = append(inbboundRules, godo.InboundRule{
			Protocol:  "tcp",
			PortRange: portString,
			Sources:   &sources,
		})
	}
	newFirewall, _, err := firewall.Create(context.TODO(), &godo.FirewallRequest{
		Name:         firewallName,
		InboundRules: inbboundRules,
	})
	if err != nil {
		log.Println("Error encountered creating DO Firewall")
		return "", err
	}
	return newFirewall.ID, err
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
// func editDOFirewall(fwID string, token string) {

// }

//Change firewall belonging to instance
func SetDOFirewall(fwID string, token string, id int) error {
	client := newDOClient(token)
	_, err := client.Firewalls.AddDroplets(context.TODO(), fwID, id)
	if err != nil {
		log.Println("Error setting the DO firewall to instance")
		return err
	}
	return nil
}
