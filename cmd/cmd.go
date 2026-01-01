package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tela "github.com/secretnamebasis/simple-tela/pkg"
)

var network string = "simulator"
var mainnet bool = false // the idea here is:
// deploy under simulation as much and as often as needed
// and when the time comes, clone a simulated deployment to mainnet
// obviously checking to make sure... is the endpoint correct
var request string = "demo-deployment"
var dURL string = "demo.tela"
var src_file string
var src_json string

func isValidTLD(tld string) bool {
	switch tld {
	case ".tela", ".dero", ".shards": // investigate other valid tld
		return true
	default:
		return false
	}
}

// the purpose of this application is to make the deployment process of tela simple
func Run() {

	for _, each := range os.Args {
		if strings.Contains(each, "=") {
			parts := strings.Split(each, "=")
			key := parts[0]
			value := parts[1]
			switch len(parts) {
			case 1:
			case 2:
				switch key {
				// case "--signature":
				// 	signature = value
				case "--durl":
					dURL = value
				case "--src-file":
					src_file = value
				case "--src-json":
					src_json = value // should be able to marshal a string at the run
				}
			default:
				fmt.Println("lol")
				return
			}
		}
	}
	// if signature == "" {
	// 	fmt.Println("must provide a signature")
	// 	return
	// }

	if dURL == "" {
		fmt.Println("must provide a durl")
		return
	}
	// let's validate the dURL
	if !isValidTLD(filepath.Ext(dURL)) {
		fmt.Println("must provide a valid top level domain name")
		return
	}

	if mainnet {
		network = "mainnet"
	}

	docs_json_data_string := "'"

	// if we are working from files, make files
	if src_file != "" && src_json == "" {
		// let's make a directory
		os.Mkdir("deployment", 0700)
		// obviously, we'll likely need some kind of xswd connection
		// typically, we are going to be using some other tool to write code, and test code
		dst := filepath.Join("deployment", time.Now().Local().Format("2006.01.02.15_04_05")) + "_" + dURL + "_" + network
		// first we need to create a new deployment
		if err := os.Mkdir(dst, 0700); err != nil {
			if !strings.Contains(err.Error(), "no such file or directory") {
				fmt.Println(err)
				return
			}
		}

		fileBytes, err := os.ReadFile(src_file)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dst, "docs.json"), fileBytes, 0644); err != nil {
			fmt.Println(err)
			return
		}

		docs_json_data_string = string(fileBytes)

	} else if src_file == "" && src_json != "" { // if we are working with strings, work from strings
		docs_json_data_string = src_json
	} else {
		fmt.Println("must provide src file or json string")
		return
	}

	var docs []tela.DOC

	if err := json.Unmarshal([]byte(docs_json_data_string), &docs); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(docs)
	tela.
}
