package cloud

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

//WriteActivityLog writes general activity to log file
func WriteActivityLog(text string) {
	//TODO Finalize a path for the activity log
	f, err := os.OpenFile("activity.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Error opening activity log file, no log will be written")
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text); err != nil {
		log.Println("Error writing activity log file, no log will be written")
	}
}

//WriteErrorLog writes application errors to log file
func WriteErrorLog(text string) bool {
	//TODO Finalize a path for the error log
	f, err := os.OpenFile("error.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Error opening error log file, no log will be written")
		return false
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text); err != nil {
		log.Println("Error writing error log file, no log will be written")
		return false
	}
	return true
}

func ParseConfig(configFile string) Config {
	var config Config
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

///////////////////

//String Slice Helper Functions//
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func removeString(s []string, e string) []string {
	for i := range s {
		if s[i] == e {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func removeDuplicateStrings(inSlice []string) (outSlice []string) {
	outSlice = inSlice[:1]
	for _, p := range inSlice {
		inOutSlice := false
		for _, q := range outSlice {
			if p == q {
				inOutSlice = true
			}
		}
		if !inOutSlice {
			outSlice = append(outSlice, p)
		}
	}
	return
}

func splitOnComma(inString string) (outSlice []string) {
	outSlice = strings.Split(inString, ",")
	return
}
