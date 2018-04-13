package misc

import (
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

//helper function to see if a file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

//misc.WriteActivityLog writes general activity to log file
func WriteActivityLog(text string) {
	usr, _ := user.Current()

	logDir := usr.HomeDir + "/.hideNsneak/log/"
	f, err := os.OpenFile(logDir+"activity.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Error opening activity log file, no log will be written: %s", err)
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text + "\n"); err != nil {
		log.Printf("Error writing activity log file, no log will be written: %s \n", err)
	}
}

//misc.WriteErrorLog writes application errors to log file
func WriteErrorLog(text string) bool {
	usr, _ := user.Current()

	logDir := usr.HomeDir + "/.hideNsneak/log/"

	f, err := os.OpenFile(logDir+"error.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Error opening error log file, no log will be written: %s \n", err)
		return false
	}

	defer f.Close()

	if _, err = f.WriteString(time.Now().UTC().Format(time.RFC850) + " : " + text + "\n"); err != nil {
		log.Printf("Error writing error log file, no log will be written: %s \n", err)
		return false
	}
	return true
}

///////////////////

//String Slice Helper Functions//
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func RemoveString(s []string, e string) []string {
	for i := range s {
		if s[i] == e {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func RemoveDuplicateStrings(inSlice []string) (outSlice []string) {
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

func SplitOnComma(inString string) (outSlice []string) {
	outSlice = strings.Split(inString, ",")
	return
}

func ValidateIntArray(integers []string) ([]int, bool) {
	var intArray []int
	for _, p := range integers {
		q, err := strconv.Atoi(p)
		intArray = append(intArray, q)
		if err != nil {
			return intArray, false
		}
	}
	return intArray, true
}

func ValidateIPArray(ips []string) bool {

	for _, ip := range ips {
		if _, _, err := net.ParseCIDR(ip); err != nil {
			if result := net.ParseIP(ip); result == nil {
				return false
			}
		}
	}
	return true
}
