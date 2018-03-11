package misc

import (
	"log"
	"os"
	"strings"
	"time"
)

//misc.WriteActivityLog writes general activity to log file
func misc.WriteActivityLog(text string) {
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

//misc.WriteErrorLog writes application errors to log file
func misc.WriteErrorLog(text string) bool {
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
