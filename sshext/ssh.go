package sshext

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh"
)

//PublicKeyFile reads a filepath to a public key file, parses it and returns it
func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading public key file")
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		fmt.Println("Error parsing public key")
		return nil
	}
	return ssh.PublicKeys(key)
}

//
func SetHomeDir(ipv4 string, username string, privateKey string) string {
	workingDir := ExecuteCmd("pwd", ipv4, username, privateKey)
	homedir := strings.TrimSpace(workingDir)
	return homedir
}

func ScpFileToHost(file string, targetDir string, username string, ipv4 string, privateKey string) bool {
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", privateKey, file, username+"@"+ipv4+":"+targetDir)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
		return false
	}
	return true
}

func ScpFileFromHost(file string, targetDir string, username string, ipv4 string, privateKey string) bool {
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4+":"+file, targetDir)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
		return false
	}
	return true
}

func RsyncDirToHost(dir string, targetDir string, username string, ipv4 string, privateKey string) error {
	bashCmd := "rsync -azu -e 'ssh  -o StrictHostKeyChecking=no -i " + privateKey + " -l " + username + "' " + dir + " " + ipv4 + ":" + targetDir
	command := exec.Command("bash", "-c", bashCmd)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Run(); err != nil {
		fmt.Printf("SCPDir failed, %s ", err)
		return err
	}
	return nil
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func RsyncFromHost(dir string, target string, username string, ipv4 string, privateKey string) error {
	rsyncCommand := "rsync -azu -e 'ssh -o StrictHostKeyChecking=no -i " + privateKey + "' " + username + "@" + ipv4 + ":" + dir + " " + target
	command := exec.Command("bash", "-c", rsyncCommand)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Start(); err != nil {
		fmt.Println("Rsync From failed")
		return err
	}
	return nil
}

func ShellSystem(ipv4 string, username string, privateKey string) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", ipv4+":22", sshConfig)
	if err != nil {
		fmt.Printf("dial failed:%v", err)
		return
	}
	session, err := conn.NewSession()
	if err != nil {
		fmt.Printf("session failed:%v", err)
		return
	}
	defer session.Close()

	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, _ := session.StdinPipe()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		fmt.Printf("request for pseudo terminal failed: %s", err)
		return
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		fmt.Printf("failed to start shell: %s", err)
		return
	}

	// Accepting commands
	for {
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		if str == "quit\n" {
			return
		}

		//TODO: Play with this a little later
		fmt.Fprint(in, str)
	}
}

func ExecuteCmd(cmd string, ipv4 string, username string, privateKey string) string {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	fmt.Println(sshConfig)

	conn, err := ssh.Dial("tcp", ipv4+":22", sshConfig)
	if err != nil {
		fmt.Printf("Error on conn statement %s", err)
	}
	session, err := conn.NewSession()
	if err != nil {
		fmt.Printf("Error on sessipn statement %s", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	return stdoutBuf.String()
}

func ExecuteBackgroundCmd(cmd string, ipv4 string, username string, privateKey string) string {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	fmt.Println(sshConfig)

	conn, _ := ssh.Dial("tcp", ipv4+":22", sshConfig)
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Start(cmd)

	return stdoutBuf.String()
}
