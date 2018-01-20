package main

import (
	// "os"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
	// "os/exec"
)

func redirectorSetup(instance Instance, teamserver string, port string) {
	sshConfig := &ssh.ClientConfig{
		User: instance.SSH.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.SSH.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	instance.executeCmd("sudo nohup apt-get update &>/dev/null & sudo nohup apt-get -y install socat &>/dev/null & sudo nohup socat TCP4-LISTEN:"+port+",fork TCP4:"+teamserver+":"+port+" &>/dev/null &", sshConfig)
}

func teamserverSetup(instance *Instance) {
	config := instance.Cloud.Config
	sshConfig := &ssh.ClientConfig{
		User: instance.SSH.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.SSH.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	instance.executeCmd(`sudo apt-get update;
		sudo apt-get install -y python-software-properties debconf-utils;
		sudo add-apt-repository -y ppa:webupd8team/java;
		sudo apt-get update;
		echo "oracle-java8-installer shared/accepted-oracle-license-v1-1 select true" | sudo debconf-set-selections;
		sudo apt-get install --no-install-recommends -y oracle-java8-installer ca-certificates;
		`, sshConfig)

	fmt.Println("Successfully installed Oracle Java")

	//TODO Change these home directories
	instance.rsyncDirToHost(config.CSDir, instance.System.HomeDir)
	fmt.Println("Copied CS")
	instance.rsyncDirToHost(config.CSProfiles, instance.System.HomeDir)
	fmt.Println("Copied Profiles")

	fmt.Println(instance.executeCmd("cd "+instance.System.HomeDir+"/"+path.Base(config.CSDir)+" && echo "+config.CSLicense+" | ./update", sshConfig))

	cmd := (`cd ` + instance.System.HomeDir + `/` + path.Base(config.CSDir) + ` && sudo ./teamserver ` + instance.Cloud.IPv4 + ` 
	` + instance.CobaltStrike.CSPassword + ` ` + instance.System.HomeDir + `/` + path.Base(config.CSProfiles) + `/
	` + instance.CobaltStrike.C2Profile + ` ` + instance.CobaltStrike.KillDate)
	instance.executeBackgroundCmd(cmd, sshConfig)
	instance.CobaltStrike.TeamserverEnabled = true
	fmt.Println("Starting teamserver")
}

func installSSLCert(instance Instance, keystore string) bool {
	config := instance.Cloud.Config
	sshConfig := &ssh.ClientConfig{
		User: instance.SSH.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.SSH.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if !(len(instance.executeCmd("which keytool", sshConfig)) > 7) {
		//Log here
		fmt.Println("keytool not in path on the target machine, check if java is installed")
		return false
	}

	sslCommand := `openssl pkcs12 -export -in ` + instance.SSL.CertLocation + `/fullchain.pem 
	-inkey ` + instance.SSL.CertLocation + `privkey.pem -out ` + instance.Cloud.Domain + `.pkcs -name ` + instance.Cloud.Domain + ` 
	-passout ` + instance.SSL.SSLKeyPass + `&& keytool -importkeystore -destskeystorepass ` + instance.SSL.SSLKeyPass + `
	 -destkeypass ` + instance.SSL.SSLKeyPass + ` -destkeystore ` + instance.Cloud.Domain + `.store -srckeystore
	  ` + instance.Cloud.Domain + `.pkcs -srcstoretype PKCS12 -srcstorepass ` + instance.SSL.SSLKeyPass + ` 
	  -alias ` + instance.Cloud.Domain

	//Log this command
	instance.executeCmd(sslCommand, sshConfig)

	if (len(instance.executeCmd("find . -maxdepth 1 -name "+instance.Cloud.Domain+".store", sshConfig))) > 5 {
		instance.executeCmd("mkdir "+instance.Cloud.Config.CSDir+"/ssl && cp "+instance.Cloud.Domain+".store "+instance.Cloud.Config.CSDir+"/ssl/", sshConfig)

	} else {
		//Log Here
		fmt.Println("Your key was not created, may have to do it manually: https://cybersyndicates.com/2016/12/egressing-bluecoat-with-cobaltstike-letsencrypt/")
		return false
	}

	return true
}

////For rewriting profiles for Domain Fronting/////
func removeHeaders(c2File string, headers []string) string {
	var matches string
	for _, header := range headers {
		re := regexp.MustCompile("\n.*header \"" + header + "\".*\";")
		matches = re.ReplaceAllString(c2File, "")
	}
	return matches
}

func fixSSLCert(c2File string, keystore string, password string) string {
	re := regexp.MustCompile("https-certificate {(.*\n)+}")
	insertCertificate := "{\n\tset keystore \"ssl/" + keystore + "\";\n\tset password \"" + password + "\";\n}"
	if re.Match([]byte(c2File)) {
		re = regexp.MustCompile("{\n.+set(.+\n)+}")
		matches := re.ReplaceAllString(c2File, insertCertificate)
		return matches
	}
	re = regexp.MustCompile(`(\#.+)+`)
	comments := re.FindString(c2File)
	editedC2File := comments + "\n\n" + insertCertificate + "\n"
	return editedC2File
}

func replaceHostHeader(c2File string, domain string) string {
	re := regexp.MustCompile("\"Host\" \".+\"")
	matches := re.ReplaceAllString(c2File, "\"Host\" \""+domain+"\"")
	return matches
}

func (instance *Instance) modifyProfile() bool {
	profileDir := instance.Cloud.Config.CSProfiles
	c2Profile := (profileDir + "/" + instance.CobaltStrike.C2Profile)
	b, err := ioutil.ReadFile(c2Profile)
	if err != nil {
		//Log here
		fmt.Println("Unable to read your C2 Profile. Make sure its in your defined profiles directory")
		return false
	}
	//This may change in the future so keep an eye on it
	problemHeaders := [...]string{"Accept-Encoding", "Connection", "Keep-Alive", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding"}
	headerSlice := problemHeaders[:]

	output := replaceHostHeader(string(b), instance.DomainFront.DomainFrontURL)

	if strings.Contains(instance.DomainFront.DomainFrontURL, "appspot") {
		output = removeHeaders(output, headerSlice)
	}
	if instance.SSL.SSLEnabled {
		output = fixSSLCert(output, "ssl/"+instance.Cloud.Domain+".store", instance.SSL.SSLKeyPass)
	}
	newProfile := profileDir + "/" + instance.Cloud.IPv4 + "-" + instance.CobaltStrike.C2Profile
	err = ioutil.WriteFile(newProfile, []byte(output), 0644)
	if err != nil {
		//log here
		fmt.Println("Unable to write to new profile, using old profile")
	} else {
		instance.CobaltStrike.C2Profile = newProfile
	}
	return true
}
