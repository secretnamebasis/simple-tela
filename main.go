package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/deroproject/derohe/walletapi/xswd"
	"github.com/gorilla/websocket"
	"github.com/secretnamebasis/simple-tela/app"
	"github.com/secretnamebasis/simple-tela/cmd"
	"golang.org/x/time/rate"
)

func main() {
	if slices.Contains(os.Args, "--gui") {
		fmt.Println("GUI ENABLED")
		app.RenderGui()
		return
	}
	fmt.Println("GUI DISABLED")
	cmd.Set_ws_conn()

	var durl, base, signed_dir string
	for _, each := range os.Args {
		switch {
		case strings.Contains(each, "="):
			kv_pair := strings.Split(each, "=")
			key := kv_pair[0]
			value := kv_pair[1]
			switch key {
			case "--durl":
				durl = value
			case "--src-directory":
				base = value
			case "--signed-files":
				signed_dir = value
			}
		default:
		}
	}
	if slices.Contains(os.Args, "--compile") {
		fp := filepath.Join(base)
		dirs, err := os.ReadDir(fp)
		if err != nil {
			fmt.Println(err)
			return
		}
		var contents = []string{}
		cmd.Walk(fp, dirs, &contents)
		code_files := []string{}
		for _, each := range contents {

			fileBytes, err := os.ReadFile(each)
			if err != nil {
				fmt.Println(err)
				continue
			}
			code_files = append(code_files, string(fileBytes))
		}
		signed_files := []string{}

		if signed_dir == "" {
			for _, each := range code_files {
				signed_files = append(signed_files, signData(each))
			}
		} else {
			dirs, err := os.ReadDir(signed_dir)
			if err != nil {
				fmt.Println(err)
				return
			}
			var contents = []string{}
			cmd.Walk(signed_dir, dirs, &contents)
			for _, each := range contents {
				fileBytes, err := os.ReadFile(each)
				if err != nil {
					fmt.Println(err)
					continue
				}
				signed_files = append(signed_files, string(fileBytes))
			}
		}
		cmd.CompileDocs(durl, fp, contents, code_files, signed_files)
		return
	}

	cmd.Run()

}

var limiter = rate.NewLimiter(9.0, 18) // 9 msg/sec, burst 18

func signData(input string) string {

	// fmt.Println(estimate)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "SignData",
		"method":  "SignData",
		"params":  []byte(input),
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	err = limiter.Wait(context.Background())
	if err != nil {
		panic(err)
	}

	err = cmd.Xswd_conn.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		panic(err)
	}

	_, msg, err := cmd.Xswd_conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	var r xswd.RPCResponse
	if err := json.Unmarshal(msg, &r); err != nil {
		return ""
	}

	return r.Result.(map[string]any)["signature"].(string)
}
