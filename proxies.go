package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// func portForwarder() {

// }
// func (instance CloudInstance) createSingleSOCKS(port string) (error) {
	
// 	return err
// }

//May have to change this back, but it should work
func (instance *CloudInstance) createSingleSOCKS(port int) {
	if !instance.SOCKSActive {
		portString := strconv.Itoa(port)
		fmt.Println(instance.Username + " " + instance.IPv4)
		instance.CMD = exec.Command("ssh", "-N", "-D", portString, "-o", "StrictHostKeyChecking=no", "-i", instance.PrivateKey, fmt.Sprintf(instance.Username+"@%s", instance.IPv4))
		stderr, err := instance.CMD.StderrPipe()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(instance.CMD)
		instance.Stderr = bufio.NewReader(stderr)
		if err := instance.CMD.Start(); err != nil {
			fmt.Println(err)
		}
		instance.SOCKSActive = true
		instance.SOCKSPort = portString
		fmt.Println("Single")
		fmt.Println(instance)
		if err != nil {
			fmt.Println("Socks Proxy Could not be created")
		}
	}
}

func createMultipleSOCKS(cloudInstances map[int]*CloudInstance, startPort int) (string, string) {
	counter := startPort
	for i := range cloudInstances {
			cloudInstances[i].createSingleSOCKS(counter)
			counter = counter + 1
		}

	proxychains := printProxyChains(cloudInstances)
	socksd := printSocksd(cloudInstances)
	return proxychains, socksd
}

func printProxyChains(cloudInstances map[int]*CloudInstance) string {
	var proxies string
	for _, c := range cloudInstances {
		if c.SOCKSActive {
			proxies = proxies + fmt.Sprintf("socks5 127.0.0.1 %s\n", c.SOCKSPort)
		}
	}
	return proxies
}

func printSocksd(cloudInstances map[int]*CloudInstance) string {
	var proxies string
	proxies = proxies + fmt.Sprintf("\"upstreams\": [\n")
	for i := range cloudInstances {
		if cloudInstances[i].SOCKSActive == true {
			proxies = proxies + fmt.Sprintf("{\"type\": \"socks5\", \"address\": \"127.0.0.1:%s\", \"target\": \"%s\"}", cloudInstances[i].SOCKSPort, cloudInstances[i].IPv4)
			if i < len(cloudInstances)-1 {
				proxies = proxies + fmt.Sprintf(",\n")
			}
		}
	}
	proxies = proxies + fmt.Sprintf("\n]\n")
	return proxies
}

//1 rewrites the proxychains file
//0 changes it back
func editProxychains(proxychainsFile string, proxies string, toggle int) {
	if toggle == 1 {
		f, err := os.OpenFile(proxychainsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Unable to open proxychains file. Please check it")
		}
		if _, err := f.Write([]byte(proxies)); err != nil {
			fmt.Println("Problem writing to proxychains file. Please check it")
		}
		if err := f.Close(); err != nil {
			fmt.Println("Problem closing proxychains file. Please check it")
		}
	} else {
		read, err := ioutil.ReadFile(proxychainsFile)
		if err != nil {
			fmt.Println("Unable to read proxychains file. Please check it")
		}
		newContents := strings.Replace(string(read), proxies, "", -1)
		err = ioutil.WriteFile(proxychainsFile, []byte(newContents), 0)
		if err != nil {
			fmt.Println("Problem rewriting old proxychains file. Please check itz")
		}
	}
}
