package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type RedirectorSource struct {
	RestrictedUA     string
	RestrictedSubnet string
	RestrictedHeader string
	DefaultRedirect  string
	C2Url            string
}

func generateSource(goSource string, fileOut string, redirector RedirectorSource) bool {
	redirectorTemplate, err := template.ParseFiles(goSource)
	if err != nil {
		//Log here
		return false
	}
	outFile, err := os.Create(fileOut)
	if err != nil {
		//Log here
		return false
	}
	err = redirectorTemplate.Execute(outFile, redirector)
	if err != nil {
		//Log here
		return false
	}
	return true
}

func generateC2Profile(c2profile string, c2out string) bool {
	b, err := ioutil.ReadFile(c2profile)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`(?s)http-get\ \{.*http-post`)
	matches := re.FindStringSubmatch(string(b))
	fmt.Printf(matches[0])
	re = regexp.MustCompile(`\n.*"Accept-Encoding".*\n`)
	matches = re.FindStringSubmatch(string(b))
	fmt.Println(strings.Split(matches[0], " "))
	re = regexp.MustCompile(`(?s)http-post\ \{.*`)
	matches = re.FindStringSubmatch(string(b))
	fmt.Printf(matches[0])
	return true
}

func createClient(projectID string, RestrictedUA string, RestrictedSubnet string, RestrictedHeader string, DefaultRedirect string, C2Url string) bool {
	redirector := RedirectorSource{
		RestrictedUA:     RestrictedUA,
		RestrictedSubnet: RestrictedSubnet,
		RestrictedHeader: RestrictedHeader,
		DefaultRedirect:  DefaultRedirect,
		C2Url:            C2Url,
	}

	return true
}
