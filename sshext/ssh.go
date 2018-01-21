package ssh

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
func setHomeDir(ipv4 string, username string, privateKey string) string {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	workingDir := ExecuteCmd("pwd", ipv4, sshConfig)
	homedir := strings.TrimSpace(workingDir)
	return homedir
}

func ScpFileToHost(file string, targetDir string, username string, ipv4 string, privateKey string) {
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", privateKey, file, username+"@"+ipv4+":"+targetDir)
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
	}
}

func ScpFileFromHost(file string, targetDir string, username string, ipv4 string, privateKey string) {
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", privateKey, username+"@"+ipv4+":"+file, targetDir)
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
	}
}

func RsyncDirToHost(file string, dir string, targetDir string, username string, ipv4 string, privateKey string) {
	command := exec.Command("rsync", "-azu", "-e", "'ssh", "-o", "StrictHostKeyChecking=no", "-i", privateKey, "-l", username+"'", dir, ipv4+":"+targetDir)
	if err := command.Start(); err != nil {
		fmt.Println("SCPDir failed")
		fmt.Println(err)
	}
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func RsyncDirFromHost(file string, dir string, targetDir string, username string, ipv4 string, privateKey string) {
	rsyncCommand := "rsync -azu -e 'ssh -o StrictHostKeyChecking=no -i " + privateKey + " -l " + username + "' " + ipv4 + ":" + dir + " " + targetDir
	command := exec.Command("bash", "-c", rsyncCommand)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Start(); err != nil {
		fmt.Println("Rsync Dir failed")
		fmt.Println(err)
	}
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
		fmt.Println("got here")
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		fmt.Println(str)
		fmt.Println(str)
		if str == "quit\n" {
			return
		}
		fmt.Fprint(in, str)
	}
}

func ExecuteCmd(cmd string, ipv4 string, config *ssh.ClientConfig) string {
	conn, _ := ssh.Dial("tcp", ipv4+":22", config)
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	return stdoutBuf.String()
}

func ExecuteBackgroundCmd(cmd string, ipv4 string, config *ssh.ClientConfig) string {
	conn, _ := ssh.Dial("tcp", ipv4+":22", config)
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Start(cmd)

	return stdoutBuf.String()
}
