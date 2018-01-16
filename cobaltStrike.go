package main

import (
	// "os"
	"path"
	"fmt"
	"golang.org/x/crypto/ssh"
	// "os/exec"
)

func (instance CloudInstance) redirectorSetup(teamserver string, port string) {
	sshConfig := &ssh.ClientConfig{
		User: instance.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
		instance.executeCmd("sudo nohup apt-get update &>/dev/null & sudo nohup apt-get -y install socat &>/dev/null & sudo nohup socat TCP4-LISTEN:"+port+",fork TCP4:"+teamserver+":"+port+" &>/dev/null &", sshConfig)
}

func (instance CloudInstance) teamserverSetup(profile string, password string, killDate string) {
	config := instance.Config
	sshConfig := &ssh.ClientConfig{
		User: instance.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	instance.executeCmd(`sudo apt-get update;
		sudo apt-get install -y python-software-properties debconf-utils;
		sudo add-apt-repository -y ppa:webupd8team/java;
		sudo apt-get update;
		echo "oracle-java8-installer shared/accepted-oracle-license-v1-1 select true" | sudo debconf-set-selections;
		sudo apt-get install --no-install-recommends -y oracle-java8-installer ca-certificates;
		sudo apt-get install -y expect`, sshConfig)

	fmt.Println("Successfully installed Oracle Java")


	//TODO Change these home directories
	instance.rsyncDirToHost(config.CSDir, instance.HomeDir)
	fmt.Println("Copied CS")
	instance.rsyncDirToHost(config.CSProfiles, instance.HomeDir)
	fmt.Println("Copied Profiles")


	fmt.Println(instance.executeCmd("cd "+ instance.HomeDir+"/"+ path.Base(config.CSDir) + " && echo " + config.CSLicense + " | ./update", sshConfig))


	fmt.Println("cd "+ instance.HomeDir+"/"+ path.Base(config.CSDir) + " && sudo ./teamserver " + instance.IPv4 + " " + password + " "+ instance.HomeDir+"/"+ path.Base(config.CSProfiles) + "/" + profile + " " + killDate, sshConfig)
	instance.executeBackgroundCmd("cd "+ instance.HomeDir+"/"+ path.Base(config.CSDir) +" && sudo ./teamserver " + instance.IPv4 + " " + password + " "+ instance.HomeDir+"/"+ path.Base(config.CSProfiles) + "/" + profile + " " + killDate, sshConfig)
	// fmt.Println("Starting teamserver")	
}

