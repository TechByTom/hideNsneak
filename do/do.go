package do

import (
	"context"
	"errors"
	"fmt"
	"log"
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

func DestroyDOInstance(token string, machineID int) bool {
	client := newDOClient(token)
	_, err := client.Droplets.Delete(context.TODO(), machineID)
	if err != nil {
		log.Printf("There was an error destroying the following machine, you may need to do cleanup:\n%d", machineID)
		return false
	}
	return true
}

//Helper method for now
func destroyMultipleDroplets(token string, droplets []godo.Droplet) {
	client := newDOClient(token)
	for _, drop := range droplets {
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

func newDropLetMultiCreateRequest(region string, client string, count int, size string, image string, fingerprint string) *godo.DropletMultiCreateRequest {
	var names []string
	for i := 0; i < count; i++ {
		name, _ := randutil.AlphaString(8)
		names = append(names, fmt.Sprintf("%s-%s", client, name))
	}
	return &godo.DropletMultiCreateRequest{
		Names:  names,
		Region: region,
		Size:   size,
		Image: godo.DropletCreateImage{
			Slug: image,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: fingerprint,
			},
		},
		Backups:           false,
		IPv6:              false,
		PrivateNetworking: false,
	}
}

func deployDO(token string, regions string, size string, image string, fingerprint string, number int, cust string) []godo.Droplet {
	var droplets []godo.Droplet
	client := newDOClient(token)
	availableRegions, err := doRegions(client)
	if err != nil {
		log.Fatalf("There was an error getting a list of regions:\nError: %s\n", err.Error())
	}

	regionCountMap, err := regionMap(availableRegions, regions, number)
	if err != nil {
		log.Fatalf("%s\n", err.Error())
	}
	for region, count := range regionCountMap {
		log.Printf("Creating %d droplets to region %s", count, region)
		drops, _, err := client.Droplets.CreateMultiple(context.TODO(), newDropLetMultiCreateRequest(region, cust, count, size, image, fingerprint))
		if err != nil {
			log.Printf("There was an error creating the droplets:\nError: %s\n", err.Error())
			return nil
		}
		droplets = append(droplets, drops...)
	}
	return droplets
}

//List existing droplets
func listDroplets(token string) []godo.Droplet {
	client := newDOClient(token)
	droplets, _, err := client.Droplets.List(context.TODO(), &godo.ListOptions{
		Page:    1,
		PerPage: 50,
	})
	if err != nil {
		log.Print("There was an error retrieving droplets.")
	}
	return droplets
}
