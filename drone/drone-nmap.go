package drone

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/lair-framework/api-server/client"
	"github.com/lair-framework/go-lair"
	"github.com/lair-framework/go-nmap"
)

const (
	version  = "2.1.1"
	tool     = "nmap"
	osWeight = 50
)

func buildProject(run *nmap.NmapRun, projectID string, tags []string) (*lair.Project, error) {
	project := &lair.Project{}
	project.ID = projectID
	project.Tool = tool
	project.Commands = append(project.Commands, lair.Command{Tool: tool, Command: run.Args})

	for _, h := range run.Hosts {
		host := &lair.Host{Tags: tags}
		if h.Status.State != "up" {
			continue
		}

		for _, address := range h.Addresses {
			switch {
			case address.AddrType == "ipv4":
				host.IPv4 = address.Addr
			case address.AddrType == "mac":
				host.MAC = address.Addr
			}
		}

		for _, hostname := range h.Hostnames {
			host.Hostnames = append(host.Hostnames, hostname.Name)
		}

		for _, p := range h.Ports {
			service := lair.Service{}
			service.Port = p.PortId
			service.Protocol = p.Protocol

			if p.State.State != "open" {
				continue
			}

			if p.Service.Name != "" {
				service.Service = p.Service.Name
				service.Product = "Unknown"
				if p.Service.Product != "" {
					service.Product = p.Service.Product
					if p.Service.Version != "" {
						service.Product += " " + p.Service.Version
					}
				}
			}

			for _, script := range p.Scripts {
				note := &lair.Note{Title: script.Id, Content: script.Output, LastModifiedBy: tool}
				service.Notes = append(service.Notes, *note)
			}

			host.Services = append(host.Services, service)
		}

		if len(h.Os.OsMatches) > 0 {
			os := lair.OS{}
			os.Tool = tool
			os.Weight = osWeight
			os.Fingerprint = h.Os.OsMatches[0].Name
			host.OS = os
		}

		project.Hosts = append(project.Hosts, *host)

	}

	return project, nil
}

func NmapImport(insecureSSL bool, limitHosts bool, forcePorts bool, filename string, lairPID string, tags string) bool {
	lairURL := os.Getenv("LAIR_API_SERVER")
	if lairURL == "" {
		log.Println("Fatal: Missing LAIR_API_SERVER environment variable")
		return false
	}
	u, err := url.Parse(lairURL)
	if err != nil {
		log.Printf("Fatal: Error parsing LAIR_API_SERVER URL. Error %s", err.Error())
		return false
	}
	if u.User == nil {
		log.Println("Fatal: Missing username and/or password")
		return false
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	if user == "" || pass == "" {
		log.Println("Fatal: Missing username and/or password")
		return false
	}
	c, err := client.New(&client.COptions{
		User:               user,
		Password:           pass,
		Host:               u.Host,
		Scheme:             u.Scheme,
		InsecureSkipVerify: insecureSSL,
	})
	if err != nil {
		log.Printf("Fatal: Error setting up client. Error %s", err.Error())
		return false
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Fatal: Could not open file. Error %s", err.Error())
		return false
	}
	hostTags := []string{}
	if tags != "" {
		hostTags = strings.Split(tags, ",")
	}
	nmapRun, err := nmap.Parse(data)
	if err != nil {
		log.Printf("Fatal: Error parsing nmap. Error %s", err.Error())
		return false
	}
	project, err := buildProject(nmapRun, lairPID, hostTags)
	if err != nil {
		log.Printf("Fatal: Error building project. Error %s", err.Error())
		return false
	}
	res, err := c.ImportProject(&client.DOptions{ForcePorts: forcePorts, LimitHosts: limitHosts}, project)
	if err != nil {
		log.Printf("Fatal: Unable to import project. Error %s", err.Error())
		return false
	}
	defer res.Body.Close()
	droneRes := &client.Response{}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Fatal: Error %s", err.Error())
		return false
	}
	if err := json.Unmarshal(body, droneRes); err != nil {
		log.Printf("Fatal: Could not unmarshal JSON. Error %s", err.Error())
		return false
	}
	if droneRes.Status == "Error" {
		log.Printf("Fatal: Import failed. Error %s", droneRes.Message)
		return false
	}
	log.Println("Success: Operation completed successfully")
	return true
}
