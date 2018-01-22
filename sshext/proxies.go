package sshext

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// func portForwarder() {

// }
// func (instance Instance) createSingleSOCKS(port string) (error) {

// 	return err
// }

//May have to change this back, but it should work
func CreateSingleSOCKS(privateKey string, username string, ipv4 string, port int) (bool, *os.Process) {
	portString := strconv.Itoa(port)
	cmd := exec.Command("ssh", "-N", "-D", portString, "-o", "StrictHostKeyChecking=no", "-i", privateKey, fmt.Sprintf(username+"@%s", ipv4))
	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return false, nil
	}
	return false, cmd.Process
}

func PrintProxyChains(socksConf map[int]string) string {
	var proxies string
	for port := range socksConf {
		proxies = proxies + fmt.Sprintf("socks5 127.0.0.1 %d\n", port)
	}
	return proxies
}

func PrintSocksd(socksConf map[int]string) string {
	proxies := fmt.Sprintf("\"upstreams\": [\n")
	for port, ip := range socksConf {
		proxies = proxies + fmt.Sprintf("{\"type\": \"socks5\", \"address\": \"127.0.0.1:%d\", \"target\": \"%s\"}", port, ip)
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
