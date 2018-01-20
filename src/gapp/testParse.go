package main

//Removed Headers
// Accept-Encoding
// Connection
// Keep-Alive
// Proxy-Authorization
// TE
// Trailer
// // Transfer-Encoding

func main() {
	b, err := ioutil.ReadFile("/Users/mike.hodges/Tools/cobalt-strike-profiles/gmail.profile")
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`(?s)http-get\ \{.*http-post`)
	matches := re.FindStringSubmatch(string(b))
	fmt.Printf(matches[0])
	re = regexp.MustCompile(`\n.*"Accept-Encoding".*\n`)
	matches = re.FindStringSubmatch(string(b))
	matches[0], " ")
	fmt.Println(strings.Split(matches[0], " "))
	// re = regexp.MustCompile(`(?s)http-post\ \{.*`)
	// matches = re.FindStringSubmatch(string(b))
	// fmt.Printf(matches[0])
}
