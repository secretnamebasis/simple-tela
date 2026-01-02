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

var appID = "6df99f80bc8b17340c21fa9c7613e9837cf641b1a1168433e8343337c752073c"
var appSig = `-----BEGIN DERO SIGNED MESSAGE-----
Address: dero1qyc96tgvz8fz623snpfwjgdhlqznamcsuh8rahrh2yvsf2gqqxdljqg9a9kka
C: d30f486cc66f6d6571112fcb3aacba4f076aba439e9bd0e84bef94b06e5c851
S: 2d839f4432e1c7a2da391dd01ed9efec64831b2bbc99a47ab4a04b283005080a

NmRmOTlmODBiYzhiMTczNDBjMjFmYTljNzYxM2U5ODM3Y2Y2NDFiMWExMTY4NDMz
ZTgzNDMzMzdjNzUyMDczYw==
-----END DERO SIGNED MESSAGE-----`
var Xswd_conn *websocket.Conn
var AppData = xswd.ApplicationData{
	Id:          appID,
	Signature:   []byte(appSig),
	Name:        "simple-tela-deploymnet-manager",
	Description: "Creating deployments on must be simple and fun! :)",
	Url:         "http://localhost:8080",
	Permissions: map[string]xswd.Permission{
		"transfer":              xswd.Ask, // ask for every transfer?
		"SignData":              xswd.AlwaysAllow,
		"GetAddress":            xswd.AlwaysAllow,
		"DERO.GetGasEstimate":   xswd.AlwaysAllow,
		"DERO.GetSC":            xswd.AlwaysAllow,
		"DERO.GetRandomAddress": xswd.AlwaysAllow,
	},
}

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
}
