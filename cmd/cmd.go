package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"

	tela "github.com/secretnamebasis/simple-tela/pkg"
)

var network string
var mainnet bool = false // the idea here is:
// deploy under simulation as much and as often as needed
// and when the time comes, clone a simulated deployment to mainnet
// obviously checking to make sure... is the endpoint correct

var ringsize = 2 // anon install will be possible, just not yet

var dURL string
var dst string
var index_scid string
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
				case "--scid":
					index_scid = value
				case "--src-file":
					src_file = value
				case "--src-json":
					src_json = value // should be able to marshal a string at the run
				case "--anon":
					ringsize = 16
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

	if network == "" {
		fmt.Println(errors.New("network is empty"))
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
		deployment := time.Now().Local().Format("2006.01.02.15_04_05")

		dst = filepath.Join("deployment", (deployment + "_" + dURL + "_" + network))
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
		fp := filepath.Join(dst, "docs.json")
		if err := os.WriteFile(fp, fileBytes, 0644); err != nil {
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

	if len(docs) == 0 {
		fmt.Println(errors.New("doc length is 0"))
		return
	}

	switch index_scid {
	case "":
		txids := []string{}
		for _, doc := range docs {

			limiter.Wait(context.Background())
			txid, err := tela.Installer(Xswd_conn, 2, doc)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(doc.NameHdr, txid)
			txs, err := tela.GetPool(Xswd_conn)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("waiting for tx to leave pool")
			for {
				txs, err = tela.GetPool(Xswd_conn)
				if err != nil {
					log.Fatal(err)
				}
				if !slices.Contains(txs, txid) {
					break
				}
				time.Sleep(time.Second)
			}
			fmt.Println("verifying transaction")

			x, _, err := tela.GetTXID(Xswd_conn, txid)
			if err != nil {
				log.Fatal(err)
			}
			b, err := hex.DecodeString(x)
			if err != nil {
				log.Fatal(err)
			}
			t := transaction.Transaction{}
			t.Deserialize(b)
			if t.Version == 0 {
				log.Fatal(err)
			}
			if getSC(txid).Code == "" {
				log.Fatal("code is empty")
			}
			doc.SCID = txid
			txids = append(txids, txid)

		}

		// now let's save those...
		if dst != "" {
			fileBytes, err := json.MarshalIndent(docs, "", " ")
			if err != nil {
				fmt.Println(err)
				return
			}
			fp := filepath.Join(dst, "docs.json")
			if err := os.WriteFile(fp, fileBytes, 0644); err != nil {
				fmt.Println(err)
				return
			}
		}
		headers := strings.Split(index_headers, ";")
		if len(headers) != 3 {
			fmt.Println(errors.New("headers are invalid"), headers)
			return
		}

		v := &tela.GetContractVersions(false)[1]
		h := tela.Headers{NameHdr: headers[0], DescrHdr: headers[1], IconHdr: headers[2]}

		index := &tela.INDEX{
			Author:    docs[0].Author,
			DURL:      dURL,
			DOCs:      txids,
			SCVersion: v,
			Headers:   h,
		}

		limiter.Wait(context.Background())
		txid, err := tela.Installer(Xswd_conn, 2, index)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("index", txid)
		index.SCID = txid

		saveIndex(index)

	default:

		fmt.Println("updating:", index_scid)

		// obviously, we are updating something
		r := getSC(index_scid)
		if r.Code == "" {
			fmt.Println(errors.New("code of index is empty"))
			return
		}

		_, _, err := tela.ValidINDEXVersion(r.Code, "")
		if err != nil {
			fmt.Println(err)
			return
		}

		current_index, err := tela.GetINDEXInfo(Xswd_conn, index_scid)
		if err != nil {
			fmt.Println(err)
			return
		}

		docs_on_file := []tela.DOC{}

		for _, each := range current_index.DOCs {
			doc, err := tela.GetDOCInfo(Xswd_conn, each)
			if err != nil {
				fmt.Println(err)
				continue
			}
			docs_on_file = append(docs_on_file, doc)
		}

		// we are trying to find out if any of the current docs satisfy the incoming changes
		doc_map := make(map[string]tela.DOC, len(docs_on_file))
		for _, each := range docs_on_file {
			start := strings.Index(each.Code, "/*")
			end := strings.Index(each.Code, "*/")

			if start == -1 || end == -1 {
				fmt.Println("could not parse multiline comment", each)
				return
			}

			doc_check := each.Code[start+2:]
			doc_check = strings.TrimSpace(strings.TrimSuffix(doc_check, "*/"))
			// in case of duplicates? //
			if _, ok := doc_map[doc_check]; !ok {
				doc_map[doc_check] = each
			}
		}

		order := []tela.DOC{}

		for _, doc := range docs {

			args, err := tela.NewInstallArgs(doc)
			if err != nil {
				log.Fatal(err)
			}

			code := args.Value(rpc.SCCODE, rpc.DataString).(string)
			if code == "" { // which it does
				continue
			}
			start := strings.Index(code, "/*")
			end := strings.Index(code, "*/")

			if start == -1 || end == -1 {
				fmt.Println("could not parse multiline comment", doc)
				return
			}

			doc_check := code[start+2:]
			doc_check = strings.TrimSpace(strings.TrimSuffix(doc_check, "*/"))

			// if the code is already in the contract... don't install again
			if document, ok := doc_map[doc_check]; ok {
				order = append(order, document)
				continue
			}

			// if it isn't on file...
			// install the document
			// txid, err := installContract(code, doc.Author, args)
			limiter.Wait(context.Background())
			txid, err := tela.Installer(Xswd_conn, 2, doc)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(doc.NameHdr, txid)

			// include its scid
			doc.SCID = txid

			// and add it to the order
			order = append(order, *doc)
		}

		if len(order) == 0 {
			fmt.Println("exited intentionally, no changes made to index")
			return
		}

		// always make doubly sure that the index is always first when present.
		corrected := []tela.DOC{}
		cutset := []tela.DOC{}
		for _, each := range order {
			switch {
			case strings.Contains(each.NameHdr, "index"):
				if strings.Contains(each.NameHdr, ".html") ||
					strings.Contains(each.NameHdr, ".php") {
					corrected = []tela.DOC{each}
					continue
				}
				fallthrough
			case !strings.Contains(each.NameHdr, "index"):
				fallthrough
			default:
				cutset = append(cutset, each)
			}
		}

		// create order from cutset
		order = cutset

		// if corrected is present
		if len(corrected) != 0 {
			corrected = append(corrected, order...)
			order = corrected
		} // now the order should be something like:
		// ['index.html','main.js','style.css','xyz.svg']
		// we aren't applying sorting

		scids := []string{}
		for _, each := range order {
			scids = append(scids, each.SCID)
		}

		// assume they are the diff scids
		diff := true
		for _, scid := range scids {
			if !slices.Contains(current_index.DOCs, scid) {
				diff = false
				break
			}
		}

		if !diff { // if they are the same... don't change
			fmt.Println("exited intentionally, no changes made to index")
			return
		}

		v := &tela.GetContractVersions(false)[1]

		h := tela.Headers{
			NameHdr:  current_index.NameHdr,
			DescrHdr: current_index.DescrHdr,
			IconHdr:  current_index.IconHdr,
		}

		index := &tela.INDEX{
			SCID:      index_scid,
			Author:    docs[0].Author,
			DURL:      dURL,
			DOCs:      scids,
			SCVersion: v,
			Headers:   h,
		}

		txid, err := tela.Updater(Xswd_conn, index)

		if err != nil {
			fmt.Println(err)
			return
		}

		if txid == "" {
			fmt.Println("failed to produce a txid")
			return
		}

		saveIndex(index)
	}

	// fmt.Println(docs)
}

func saveIndex(index *tela.INDEX) {
	if index == nil {
		return
	}
	if dst != "" {
		fileBytes, err := json.MarshalIndent(index, "", " ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fp := filepath.Join(dst, "index.json")
		if err := os.WriteFile(fp, fileBytes, 0644); err != nil {
			fmt.Println(err)
			return
		}
	}
}
