package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"

	tela "github.com/secretnamebasis/simple-tela/pkg"
)

var network string = "simulator"
var mainnet bool = false // the idea here is:
// deploy under simulation as much and as often as needed
// and when the time comes, clone a simulated deployment to mainnet
// obviously checking to make sure... is the endpoint correct
var request string = "demo-deployment"
var dURL string = "demo.tela"
var dst string
var index_headers string
var src_file string
var src_json string

// the purpose of this application is to make the deployment process of tela simple
func Run() {
	if Xswd_conn == nil {
		if err := Set_ws_conn(); err != nil {
			fmt.Println(err)
			return
		}
	}

	for _, each := range os.Args {
		if strings.Contains(each, "=") {
			parts := strings.Split(each, "=")
			key := parts[0]
			value := parts[1]
			switch len(parts) {
			case 1:
			case 2:
				switch key {
				case "--headers":
					// just in case
					value = strings.TrimPrefix(value, `"`)
					value = strings.TrimSuffix(value, `"`)

					index_headers = value
				case "--durl":
					dURL = value
				case "--network":
					network = value
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

	// because these aren't initialized anywhere
	fmt.Println(network)
	switch network {
	case "mainnet":
		globals.Arguments["--testnet"] = false
		globals.Arguments["--simulator"] = false
	case "testnet":
		globals.Arguments["--testnet"] = true
		globals.Arguments["--simulator"] = false
	case "simulator":
		globals.Arguments["--testnet"] = true
		globals.Arguments["--simulator"] = true
	}
	globals.InitNetwork()

	// if mainnet {
	// 	network = "mainnet"
	// }

	docs_json_data_string := "'"

	// if we are working from files, make files
	if src_file != "" && src_json == "" {
		// let's make a directory
		os.Mkdir("deployment", 0700)
		// obviously, we'll likely need some kind of xswd connection
		// typically, we are going to be using some other tool to write code, and test code
		dst = filepath.Join("deployment", time.Now().Local().Format("2006.01.02.15_04_05")) + "_" + dURL + "_" + network
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

	var docs []*tela.DOC

	if err := json.Unmarshal([]byte(docs_json_data_string), &docs); err != nil {
		fmt.Println(err)
		return
	}
	txids := []string{}
	for _, each := range docs {
		args, err := tela.NewInstallArgs(each)
		if err != nil {
			log.Fatal(err) // I guess I answered my own question
			// if the file is too large, we have to apply compression... but let's not apply compression at run time
			// let's do it at compiling
		}
		if code := args.Value(rpc.SCCODE, rpc.DataString).(string); code != "" {

			txid, err := installContract(code, each.Author, args)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(txid)
			each.SCID = txid
			txids = append(txids, txid)
		}
	}
	// now let's save those...
	if dst != "" {
		fileBytes, err := json.MarshalIndent(docs, "", " ")
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dst, "docs.json"), fileBytes, 0644); err != nil {
			fmt.Println(err)
			return
		}
	}
	headers := strings.Split(index_headers, ";")
	if len(headers) != 3 {
		fmt.Println(errors.New("headers are invalid"), headers)
		return
	}
	nameHdr := headers[0]
	descHdr := headers[1]
	iconHdr := headers[2]
	index := &tela.INDEX{
		Author:    docs[0].Author,
		DURL:      dURL,
		DOCs:      txids,
		SCVersion: &tela.GetContractVersions(false)[1],
		Headers:   tela.Headers{NameHdr: nameHdr, DescrHdr: descHdr, IconHdr: iconHdr},
	}
	args, err := tela.NewInstallArgs(index)
	if err != nil {
		log.Fatal(err) // I guess I answered my own question
		// if the file is too large, we have to apply compression... but let's not apply compression at run time
		// let's do it at compiling
	}

	if code := args.Value(rpc.SCCODE, rpc.DataString).(string); code != "" {
		txid, err := installContract(code, index.Author, args)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(txid)
		index.SCID = txid
	}

	if dst != "" {

		fileBytes, err := json.MarshalIndent(index, "", " ")
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dst, "index.json"), fileBytes, 0644); err != nil {
			fmt.Println(err)
			return
		}
	}

	// fmt.Println(docs)
}
