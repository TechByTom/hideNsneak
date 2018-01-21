package cobaltstrike

import (
	// "os"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
	// "os/exec"
	ssh-etf "github.com/rmikehodges/SneakyVulture/ssh"
)

func redirectorSetup(privateKey string, teamserver string, port string, username string) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	ssh.ExecuteCmd(`sudo nohup apt-get update &>/dev/null & sudo nohup apt-get -y install socat 
		&>/dev/null & sudo nohup socat TCP4-LISTEN:`+port+`
		,fork TCP4:`+teamserver+`:`+port+` &>/dev/null &`, sshConfig)
}
func teamserverSetup(privateKey string, homeDir string, csdir string, homedir string, csprofiles string) {
	config := instance.Cloud.Config
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	instance.ExecuteCmd(`sudo apt-get update;
		sudo apt-get install -y python-software-properties debconf-utils;
		sudo add-apt-repository -y ppa:webupd8team/java;
		sudo apt-get update;
		echo "oracle-java8-installer shared/accepted-oracle-license-v1-1 select true" | sudo debconf-set-selections;
		sudo apt-get install --no-install-recommends -y oracle-java8-installer ca-certificates;
		`, sshConfig)

	fmt.Println("Successfully installed Oracle Java")

	//TODO Change these home directories
	instance.RsyncDirToHost(csdir, homedir)
	fmt.Println("Copied CS")
	instance.RsyncDirToHost(csprofiles, homedir)
	fmt.Println("Copied Profiles")

	fmt.Println(ssh.ExecuteCmd("cd "+homedir+"/"+path.Base(csdir)+" && echo "+cslicense+" | ./update", sshConfig))

	cmd := (`cd ` + homedir + `/` + path.Base(csdir) + ` && sudo ./teamserver ` + ipv4 + ` 
	` + instance.CobaltStrike.CSPassword + ` ` + homedir + `/` + path.Base(csprofiles) + `/
	` + c2profile + ` ` + instance.CobaltStrike.KillDate)
	ssh.ExecuteBackgroundCmd(cmd, sshConfig)
	instance.CobaltStrike.TeamserverEnabled = true
	fmt.Println("Starting teamserver")
}

func installSSLCert(username string, domain string, certLocation string, sslKeyPass string, csdir string, privateKey string, keystore string) bool {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if !(len(instance.executeCmd("which keytool", sshConfig)) > 7) {
		//Log here
		fmt.Println("keytool not in path on the target machine, check if java is installed")
		return false
	}

	sslCommand := `openssl pkcs12 -export -in ` + certLocation + `/fullchain.pem 
	-inkey ` + certLocation + `privkey.pem -out ` + domain + `.pkcs -name ` + domain + ` 
	-passout ` + sslKeyPass + `&& keytool -importkeystore -destskeystorepass ` + sslKeyPass + `
	 -destkeypass ` + sslKeyPass + ` -destkeystore ` + domain + `.store -srckeystore
	  ` + domain + `.pkcs -srcstoretype PKCS12 -srcstorepass ` + sslKeyPass + ` 
	  -alias ` + domain

	//Log this command
	instance.executeCmd(sslCommand, sshConfig)

	if (len(instance.executeCmd("find . -maxdepth 1 -name "+domain+".store", sshConfig))) > 5 {
		instance.executeCmd("mkdir "+csdir+"/ssl && cp "+domain+".store "+csdir+"/ssl/", sshConfig)

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

func modifyProfile(ipv4 string, csProfileDir string, c2profile string, domainFrontURL string, sslKeyPass string) bool {
	profileDir := csProfileDir
	c2Profile := (profileDir + "/" + c2profile)
	b, err := ioutil.ReadFile(c2Profile)
	if err != nil {
		//Log here
		fmt.Println("Unable to read your C2 Profile. Make sure its in your defined profiles directory")
		return false
	}
	//This may change in the future so keep an eye on it
	problemHeaders := [...]string{"Accept-Encoding", "Connection", "Keep-Alive", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding"}
	headerSlice := problemHeaders[:]

	output := replaceHostHeader(string(b), domainFrontURL)

	if strings.Contains(domainFrontUrl, "appspot") {
		output = removeHeaders(output, headerSlice)
	}
	if instance.SSL.SSLEnabled {
		output = fixSSLCert(output, "ssl/"+domain+".store", sslKeyPass)
	}
	newProfile := profileDir + "/" + ipv4 + "-" + c2profile
	err = ioutil.WriteFile(newProfile, []byte(output), 0644)
	if err != nil {
		//log here
		fmt.Println("Unable to write to new profile, using old profile")
	} else {
		c2profile = newProfile
	}
	return true
}
