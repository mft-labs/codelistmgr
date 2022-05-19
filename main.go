package main

import (
	"flag"
	"fmt"
	amf_crypto "github.com/mft-labs/amf_crypto"
	"gopkg.in/ini.v1"
	"os"
)

var errorsList = make([]string, 0)

func loadConfig(fname string) *ini.File {
	var newconfig *ini.File
	var err error
	newconfig, err = ini.Load(fname)
	if err != nil {
		errorsList = append(errorsList, "ERROR: invalid config file or missing DEFAULT section")
		showErrors("")
		os.Exit(10002)
	}
	return newconfig
}

func encrypt(text string) string {
	encrypted, err := amf_crypto.Encrypt(text)
	if err != nil {
		return ""
	}
	return encrypted
}

func main() {
	var conf string
	var input string
	//var action string
	//flag.StringVar(&action, "action", "", "action (encrypt/bulkupdate)")
	flag.StringVar(&input, "input", "", "input file name")
	flag.StringVar(&conf, "conf", "", "configuration file name")
	flag.Parse()

	if conf == "" && input == "" {
		showUsage()
		os.Exit(10001)
	}

	if conf == "" {
		conf = "apimgr.conf"
	}
	if input == "" {
		errorsList = append(errorsList, "Missing input document")
	}
	validateInputs(conf, input)
	if len(errorsList) == 0 {
		manageBulkUpdate(conf, input)
	} else {
		showErrors("")
		os.Exit(10001)
	}

}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	} else {
		return false
	}
}

func validateInputs(conf, infile string) {
	if !fileExists(conf) {
		errorsList = append(errorsList, conf+" not found")
	}
	if infile != "" {
		if !fileExists(infile) {
			errorsList = append(errorsList, infile+" not found")
		}
	}
}

func manageBulkUpdate(conf, infile string) {
	service := &apiMgr{}
	service.infile = infile
	service.config = loadConfig(conf)
	service.errorsList = make([]string, 0)
	err := service.init()
	if err == nil {
		err = service.runUpdate()
		if err != nil {
			errorsList = service.errorsList
			showErrors("ERROR: CodeList update failed")
			os.Exit(10003)
		}
	} else {
		errorsList = service.errorsList
		showErrors("ERROR: Missing keys or DEFAULT section in config file")
		os.Exit(10002)
	}

}

func showUsage() {
	fmt.Println("AMF CodeList Manager")
	fmt.Println("======================================================================")
	fmt.Printf("Invalid request\n\n")
	fmt.Println("Usage:")
	fmt.Printf("%s [-conf <config filename>] -input <input XLSX document>\n", os.Args[0])
	fmt.Printf("\nconfiguration file is optional, apimgr.conf is assumed as the default configuration file.")
}

func showErrors(title string) {
	fmt.Println("AMF CodeList Manager")
	fmt.Println("======================================================================")
	fmt.Println("Please fix the following errors and run again.")
	if title != "" {
		fmt.Printf("%s\n", title)
	}
	for _, errin := range errorsList {
		fmt.Println(errin)
	}
}
