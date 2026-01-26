package cmd

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi/xswd"
	"github.com/gorilla/websocket"
	tela "github.com/secretnamebasis/simple-tela/pkg"
	"golang.org/x/time/rate"
)

func ignore(name string) bool {
	switch {
	case strings.Contains(name, ".keep"):
		return true
	default:
		return false
	}
}

func isValidTLD(tld string) bool {
	switch tld {
	case ".tela", ".dero", ".shards": // investigate other valid tld
		return true
	default:
		return false
	}
}

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
		"transfer":              xswd.AlwaysAllow, // ask for every transfer?
		"SignData":              xswd.AlwaysAllow,
		"GetAddress":            xswd.AlwaysAllow,
		"DERO.GetGasEstimate":   xswd.AlwaysAllow,
		"DERO.GetSC":            xswd.AlwaysAllow,
		"DERO.GetRandomAddress": xswd.AlwaysAllow,
	},
}
var websocket_endpoint string = "ws://127.0.0.1:44326/xswd"

func Set_ws_conn() error {

	for _, each := range os.Args {
		if !strings.Contains(each, "--ws-address=") {
			continue
		}
		endpoint := strings.Split(each, "=")[1]
		websocket_endpoint = "ws://" + endpoint + "/xswd"
	}

	var err error

	fmt.Printf("Connecting to %s\n", websocket_endpoint)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // allow self-signed certs
	}

	conn, _, err := dialer.Dial(websocket_endpoint, nil)
	Xswd_conn = conn

	if err != nil {
		return err
	}
	fmt.Println("WebSocket xswd_connected")

	if err := Xswd_conn.WriteJSON(AppData); err != nil {
		panic(err)
	}
	fmt.Println("Auth handshake sent")

	_, msg, err := Xswd_conn.ReadMessage()
	if err != nil {
		return err
	}

	var res xswd.AuthorizationResponse
	if err := json.Unmarshal(msg, &res); err != nil {
		return err
	}

	if !res.Accepted {
		return (errors.New("app not accepted"))
	}

	return nil
}

// going to want some consistency for how to walk
func Walk(name string, paths []os.DirEntry, contents *[]string) {
	// fmt.Println(paths)
	for _, each := range paths {
		fp := filepath.Join(name, each.Name())
		if ignore(each.Name()) {
			continue
		}
		if each.IsDir() {
			entries, err := os.ReadDir(fp)
			if err != nil {
				fmt.Println(err)
				return
			}
			Walk(fp, entries, contents)
		} else {
			//
			*contents = append(*contents, fp)
		}
	}
}

func CompileDocs(dURL, base string, contents []string, code, signed_code []string) (docs []tela.DOC) {
	for i, each := range contents {

		name := strings.TrimPrefix(each, base)

		// with windows, the trailing slash \ is used by the fs
		// with everyone else, a leading slash / is the case
		// convert to slash element regardless
		name = filepath.ToSlash(name)

		// now remove the first element, the root
		name = strings.TrimPrefix(name, "/")

		subdir := ""
		// we are going to make an assumption that a file this/file.txt doesn't work
		// and neither does it work to use the trailing slash, \ , eg this\file.txt
		// thus, if the name of the contents contains a leading slash, it is a subdir
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")

			// remove the final part as it is the base, or filename
			dirs := parts[:(len(parts) - 1)]

			// make the subdirs
			subdir = strings.Join(dirs, "/")

			// the base is now the name
			name = filepath.Base(name)
		}

		docType := tela.ParseDocType(name)

		// fmt.Printf("doc-type: %13s file-ext: %7s file-name: %s\n", docType, fileEx, name)

		b, err := base64.StdEncoding.DecodeString(signed_code[i])
		if err != nil {
			fmt.Println(err)
			return
		}
		address, c_value, s_value, err := tela.ParseSignature(b)

		// capture the code in question
		c := code[i]

		// let's determine compression
		compression := ""

		// we need the size of the code
		if tela.GetCodeSizeInKB(c) >= tela.MAX_DOC_CODE_SIZE {
			// there are 2 strategies for dealing with the ceiling:
			// 	- compress
			// 	- shard

			// we take an opinionated approach and just compress
			// but it would seem as though the compression is secondary to sharding
			// see CreateShardFiles
			// additionally, CreateShardFiles only compresses if and only if we are reading from file
			// thus, we must compress or it doesn't happen
			// again, how would we know that the user wants the files compressed instead of sharding...
			// we don't - we take an opinionated approach and simply
			// apply compression
			c, err = tela.Compress([]byte(c), tela.COMPRESSION_GZIP)
			if err != nil {
				fmt.Println(err)
				continue
			}

			// maybe this is intentional?
			// if c == "" {
			// 	fmt.Println(errors.New("code is empty"))
			// 	continue
			// }

			compression = tela.COMPRESSION_GZIP
			name += tela.COMPRESSION_GZIP

		}

		// there is only 1 version of the doc contract at the moment
		version := &tela.GetContractVersions(true)[0]

		// if still too big
		// make docs for each one
		if tela.GetCodeSizeInKB(c) >= tela.MAX_DOC_CODE_SIZE {

			// the next process would be to shard
			// if err := tela.CreateShardFiles(name, compression); err != nil {
			// 	fmt.Println(err)
			// 	continue
			// } // but let's not use that entire process, we have no intention of writing files to disc yet.
			// so let's do some copy pasta magic
			if !utf8.ValidString(c) {
				err = fmt.Errorf("cannot shard file %s", name)
			}
			content := []byte(c)
			total, size := tela.GetTotalShards(content)
			fmt.Println(total, size)

			// at this point, compression is already added
			newFileName := func(i int, name, ext string) string {
				return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(name, ext), i, ext)
			}

			ext := filepath.Ext(name)

			count := 0
			for start := int64(0); start < size; start += tela.SHARD_SIZE {
				end := start + tela.SHARD_SIZE
				if end > size {
					end = size
				}

				count++
				shardName := newFileName(count, name, ext)

				codeShard := string(content[start:end])

				// hold on now... we have to sign all of these?
				doc := tela.DOC{
					Code:    codeShard,
					DocType: docType,
					DURL:    dURL + tela.TAG_DOC_SHARD, // N.B.
					// There is no where in TELA where `.shard` is used
					// There is, however, somewhere where `.shards` is used,
					//
					Headers: tela.Headers{
						NameHdr: shardName,
						// DescrHdr: ,
					},
					SubDir:      subdir,
					Compression: compression,
					// we'll just assume it is all apart of the same signature
					// or should the code go back up for signing?
					Signature: tela.Signature{CheckC: c_value, CheckS: s_value},
					SCVersion: version,
					Author:    address,
				}
				docs = append(docs, doc)
			}
		} else {
			// I guess we could make a table to input all the data
			doc := tela.DOC{
				Code:    c, // this is the contents of the file, but it gets re-written at install
				DocType: docType,
				DURL:    dURL, // this is tricky because this is a name-space thing...
				// in tela, .shards is a valid tld

				SubDir:      subdir,      // this is tricky as well because it is a routing thing
				Compression: compression, // this really isn't all that tricky, are we compressing the data?
				// what makes it tricky is the concept of sharding...
				Headers: tela.Headers{
					NameHdr: name, // On-chain name of SC. For TELA-DOCs, they are recreated using this as the file name, it should include the file extension
					// DescrHdr: , // unfortunately, how are we supposed to get this?
					// IconHdr: , // and what about this one... how are we supposed to handle this?
				},
				Signature: tela.Signature{CheckC: c_value, CheckS: s_value},
				SCVersion: version,
				Author:    address,
			}
			docs = append(docs, doc)
		}
	}
	// order matters... the index is a priority document
	corrected := []tela.DOC{}
	cutset := []tela.DOC{}
	for _, each := range docs {
		if !strings.Contains(each.NameHdr, "index") {
			cutset = append(cutset, each)
			continue
		}
		if !strings.Contains(each.NameHdr, ".html") && !strings.Contains(each.NameHdr, ".php") {
			cutset = append(cutset, each)
			continue
		}
		// should always be first
		corrected = []tela.DOC{each}
	}

	docs = cutset
	if len(corrected) > 0 {
		docs = append(corrected, docs...)
	}

	fmt.Println("length of docs", len(docs))
	jsonBytes, err := json.MarshalIndent(docs, "", " ")
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := os.Mkdir("src", 0700); err != nil {
		if !os.IsExist(err) {
			fmt.Println(err)
			return
		}
	}
	if err := os.WriteFile(filepath.Join("src", "docs.json"), jsonBytes, 0644); err != nil {
		fmt.Println(err)
		return
	}

	return
}

var fee_buffer = uint64(181)

// Do not over-crowd the websocket
var limiter = rate.NewLimiter(9.0, 18) // 9 msg/sec, burst 18
func getGasEstimate(payload map[string]any) rpc.GasEstimate_Result {

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Failed to marshal payload:", err)
		return rpc.GasEstimate_Result{}
	}

	type response struct {
		Result rpc.GasEstimate_Result
	}
	var r response
	if err := json.Unmarshal(postBytes(jsonBytes), &r); err != nil {
		fmt.Println("Failed to unmarshal response:", err)
		return rpc.GasEstimate_Result{}
	}
	return r.Result
}
func getRandAddr() string {
	jsonBytes, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      "RANDOM ADDRES",
		"method":  "DERO.GetRandomAddress",
	})
	if err != nil {
		fmt.Println("Failed to marshal payload:", err)
		return ""
	}

	var r xswd.RPCResponse
	// fmt.Println(string(respBody))
	if err := json.Unmarshal(postBytes(jsonBytes), &r); err != nil {
		fmt.Println("Failed to unmarshal:", err)
		return ""
	}
	return r.Result.(map[string]any)["address"].([]any)[0].(string)
}
func installContract(code, address string, args rpc.Arguments) (string, error) {
	addr := getRandAddr()

	// // fmt.Println(network, a)
	// // try_again:
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "GAS ESTIMATE",
		"method":  "DERO.GetGasEstimate",
		"params": rpc.GasEstimate_Params{
			Transfers: []rpc.Transfer{
				{
					SCID:        crypto.ZEROHASH,
					Destination: addr,
					Amount:      0,
					Burn:        0,
				},
			},
			SC_Code:  code,
			Ringsize: 2,
			SC_Value: 0,
			Signer:   address,
			SC_RPC:   args,
		}}

	estimate := getGasEstimate(payload)
	if estimate.GasCompute == 0 && estimate.GasStorage == 0 {
		log.Fatal("no gas", payload, estimate)
	}
	val1 := estimate.GasStorage
	if args.HasValue(rpc.SCACTION, rpc.DataUint64) && args.Value(rpc.SCACTION, rpc.DataUint64).(uint64) == uint64(rpc.SC_CALL) {
		val1 += estimate.GasCompute
	}
	payload = map[string]any{
		"jsonrpc": "2.0",
		"id":      "CONTRACT INSTALL",
		"method":  "transfer",
		"params": rpc.Transfer_Params{
			Transfers: []rpc.Transfer{
				{
					Destination: getRandAddr(),
					Amount:      0,
					Burn:        0,
				},
			},
			SC_Code:  code,
			Ringsize: 2,
			Signer:   address,
			Fees:     val1 + fee_buffer,
			SC_RPC:   args,
		},
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var r xswd.RPCResponse
	if err := json.Unmarshal(postBytes(jsonBytes), &r); err != nil {
		return "", err
	}
	if r.Result == nil {
		return "", errors.New("result is empty")
	}
	if r.Result.(map[string]any)["txid"].(string) == "" {
		return "", errors.New("txid is blank")
	}
	scid := r.Result.(map[string]any)["txid"].(string)
	return scid, nil
	// if !inPool(scid) {
	// time.Sleep(target)
	// }

	// sc := getSC(scid)
	// if sc.Code == codeSwitch(path) {
	// 	fmt.Println("path", path, "txid", scid)
	// 	return scid, nil
	// }
	// goto try_again // unfortunately, that means we sometimes install twice
}
func postBytes(b []byte) []byte {
	err := limiter.Wait(context.Background())
	if err != nil {
		panic(err)
	}

	err = Xswd_conn.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		panic(err)
	}

	_, msg, err := Xswd_conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	return msg
}

func getSC(scid string) rpc.GetSC_Result {

	type response struct {
		Result rpc.GetSC_Result
	}
	var r response
	// fmt.Println(string(body))
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "GET SC",
		"method":  "DERO.GetSC",
		"params": rpc.GetSC_Params{
			SCID:      scid,
			Code:      true,
			Variables: true,
		},
	}
	byt, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload", err)
		return rpc.GetSC_Result{}
	}
	if err := json.Unmarshal(postBytes(byt), &r); err != nil {
		fmt.Println("Failed to unmarshal response:", err)
		return rpc.GetSC_Result{}
	}
	return r.Result
}
