package google

import (
	"fmt"
	"io/ioutil"
	"regexp"
)

//This file should be available to the Cloudfront files as well

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
	if re.Match([]byte(c2File)) {
		insertCertificate := "{\n\tset keystore \"ssl/" + keystore + "\";\n\tset password \"" + password + "\";\n}"
		re = regexp.MustCompile("{\n.+set(.+\n)+}")
		matches := re.ReplaceAllString(c2File, insertCertificate)
		return matches
	}
	return ""
}

func replaceHostHeader(c2File string, domain string) string {
	re := regexp.MustCompile("\"Host\" \".+\"")
	matches := re.ReplaceAllString(c2File, "\"Host\" \""+domain+"\"")
	return matches
}

func generateC2Profile(c2profile string, c2out string, keystore string, password string, ssl bool, domain string) bool {
	problemHeaders := [...]string{"Accept-Encoding", "Connection", "Keep-Alive", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding"}
	headerSlice := problemHeaders[:]
	b, err := ioutil.ReadFile(c2profile)
	if err != nil {
		fmt.Println("Unable to open file for editing")
		return false
	}
	fileString := string(b)
	fileString = removeHeaders(fileString, headerSlice)
	if ssl {
		fileString = fixSSLCert(fileString, keystore, password)
	}
	fileString = replaceHostHeader(fileString, domain)
	err = ioutil.WriteFile(c2out, []byte(fileString), 0644)
	if err != nil {
		fmt.Println("Unable to write new C2 file")
		return false
	}
	return true
}
