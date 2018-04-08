package google

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type RedirectorSource struct {
	RestrictedUA     string
	RestrictedSubnet string
	RestrictedHeader string
	DefaultRedirect  string
	HeaderName       string
	HeaderValue      string
	C2Url            string
}

const appYaml = `runtime: go
api_version: go1

handlers:
- url: /.*
  script: _go_app`

const redirectorString = `package goengine
import (
	"net"
	"net/http"
	"net/url"

	"github.com/rmikehodges/gaereverseproxy"
)

type Prox struct {
	// target url of reverse proxy
	target *url.URL
	// instance of Go ReverseProxy thatwill do the job for us
	proxy *gaereverseproxy.ReverseProxy
}

func validUA(userAgent string) bool {
	ua := "{{.RestrictedUA}}"
	if ua != "" && ua != userAgent {
		return false
	}
	return true
}

func validIP(remoteIP string) bool {
	subnet := "{{.RestrictedSubnet}}"
	if len(subnet) > 8 {
		_, cidr, err := net.ParseCIDR(subnet)
		if err != nil {
			return false
		}
		if !cidr.Contains(net.ParseIP(remoteIP)) {
			return false
		}
	}
	return true
}

//TODO: Add header templating
func validHeader(remoteHeader http.Header) bool {
	header, value := "{{.HeaderName}}", "{{.HeaderValue}}"
	if header != "" && value != "" {
		if remoteHeader.Get(header) != value {
			return false
		}
	}
	return true
}

// small factory
func New(target string) *Prox {
	url, _ := url.Parse(target)
	// you should handle error on parsing
	return &Prox{target: url, proxy: gaereverseproxy.NewSingleHostReverseProxy(url)}
}

func (p *Prox) handle(w http.ResponseWriter, r *http.Request) {
	// call to magic method from ReverseProxy object
	if !validUA(r.UserAgent()) || !validIP(r.RemoteAddr) || !validHeader(r.Header) {
		http.Redirect(w, r, "{{.DefaultRedirect}}", 301)
	} else {
		p.proxy.ServeHTTP(w, r)
	}
}

func init() {
	proxy := New("{{.C2Url}}")

	// server
	http.HandleFunc("/", proxy.handle)
}

`

func generateSource(projectDir string, redirector RedirectorSource) bool {
	redirectorTemplate, err := template.New("source-code").Parse(redirectorString)
	if err != nil {
		//Log here
		log.Printf("Error Parsing template string: %s", err)
		return false
	}
	if _, err := os.Stat(projectDir + "/redirector.go"); err == nil {
		err := os.Remove(projectDir + "/redirector.go")
		if err != nil {
			fmt.Println("There was a problem deleting the previous Go file. It may not have updated correctly")
			return false
		}
	}
	outFile, err := os.Create(projectDir + "/redirector.go")
	if err != nil {
		//Log here
		log.Printf("Error creating Go file: %s", err)
		return false
	}
	defer outFile.Close()
	err = redirectorTemplate.Execute(outFile, redirector)
	if err != nil {
		//Log here
		log.Printf("Error marshalling redirector object to template file: %s", err)
		return false
	}
	appFile, err := os.Create(projectDir + "/app.yaml")
	if err != nil {
		log.Println("Error creating app.yaml")
		return false
	}
	defer appFile.Close()
	_, err = appFile.Write([]byte(appYaml))
	if err != nil {
		log.Println("Error writing app.yaml")
		return false
	}
	return true
}

func createClient(projectName string, RestrictedUA string, RestrictedSubnet string, RestrictedHeader string,
	DefaultRedirect string, C2Url string, projectDir string) bool {
	redirector := RedirectorSource{
		RestrictedUA:     RestrictedUA,
		RestrictedSubnet: RestrictedSubnet,
		RestrictedHeader: RestrictedHeader,
		DefaultRedirect:  DefaultRedirect,
		C2Url:            C2Url,
	}
	if !generateSource(projectDir, redirector) {
		return false
	}
	return true
}

func execGCloud(newProject bool, projectName string, projectDir string) bool {
	var outb bytes.Buffer
	cmd := exec.Command("gcloud", "-v")
	cmd.Stdout = &outb
	err := cmd.Run()
	if err != nil {
		fmt.Println("Unable to verify if gcloud is installed..returning")
		return false
	}
	output := outb.String()
	if len(strings.Split(output, "\n")) < 2 {
		fmt.Println("gcloud doesn't appear to be installed")
		return false
	}
	if newProject {
		cmd := exec.Command("gcloud", "projects", "create", projectName)
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error creating project: %s", err)
			return false
		}
	}
	cmd = exec.Command("gcloud", "app", "deploy", projectDir+"/app.yaml", "--project", projectName)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error creating project: %s", err)
		return false
	}
	return true
}

func CreateRedirector(projectName string, RestrictedUA string, RestrictedSubnet string, RestrictedHeader string,
	DefaultRedirect string, C2Url string, newProject bool, projectDir string, c2Profile string, c2Out string,
	keystore string, keyStorepass string) (bool, string) {
	ssl := false
	parsedURL, err := url.Parse(C2Url)
	if err != nil {
		fmt.Println("Invalid URL was passed")
		return false, ""
	}
	if !createClient(projectName, RestrictedUA, RestrictedSubnet, RestrictedHeader, DefaultRedirect, C2Url, projectDir) {
		fmt.Println("There was a problem generating the Go source code")
		return false, ""
	}
	if !execGCloud(newProject, projectName, projectDir) {
		fmt.Println("There was a problem during the gcloud upload")
		return false, ""
	}

	if parsedURL.Scheme == "https" {
		ssl = true
	}
	if !generateC2Profile(c2Profile, c2Out, keystore, keyStorepass, ssl, parsedURL.Hostname()) {
		fmt.Println("There was an issue rewriting the C2 profile. You will have to do so manually")
	}
	return true, "https://" + projectName + ".appspot.com"
}
