package google

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

//Removed Headers
// Accept-Encoding
// Connection
// Keep-Alive
// Proxy-Authorization
// TE
// Trailer
// // Transfer-Encoding

func parse() {
	b, err := ioutil.ReadFile("/Users/mike.hodges/Tools/cobalt-strike-profiles/gmail.profile")
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`(?s)http-get\ \{.*http-post`)
	matches := re.FindStringSubmatch(string(b))
	fmt.Printf(matches[0])
	re = regexp.MustCompile(`\n.*"Accept-Encoding".*\n`)
	matches = re.FindStringSubmatch(string(b))
	fmt.Println(strings.Split(matches[0], " "))
	// re = regexp.MustCompile(`(?s)http-post\ \{.*`)
	// matches = re.FindStringSubmatch(string(b))
	// fmt.Printf(matches[0])
}
