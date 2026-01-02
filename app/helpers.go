package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deroproject/derohe/walletapi/xswd"
	"github.com/gorilla/websocket"
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

// going to want some consistency for how to walk
func Walk(name string, paths []os.DirEntry, contents *[]string) {
	// fmt.Println(paths)
	for _, each := range paths {
		fp := filepath.Join(name, each.Name())
		if ignore(each.Name()) {
			continue
		}
		fmt.Println(fp)
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

var gi_conn *websocket.Conn
var ws = "127.0.0.1:9190"

func set_gnomon_conn() error {

	url := "ws://" + ws + "/ws"
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true, // allow self-signed certs
	}}
	var err error
	gi_conn, _, err = dialer.Dial(url, nil)
	if err != nil {
		return err
	}

	return nil
}

func getDaemonEndpoint() xswd.GetDaemon_Result {

	// fmt.Println(estimate)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "GetDaemon",
		"method":  "GetDaemon",
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return xswd.GetDaemon_Result{}
	}

	err = xswd_conn.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		panic(err)
	}

	_, msg, err := xswd_conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	var r xswd.RPCResponse
	if err := json.Unmarshal(msg, &r); err != nil {
		return xswd.GetDaemon_Result{}
	}

	raw, err := json.Marshal(r.Result)
	if err != nil {
		return xswd.GetDaemon_Result{}
	}

	result := xswd.GetDaemon_Result{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return xswd.GetDaemon_Result{}
	}

	return result

}

// Do not over-crowd the websocket
var limiter = rate.NewLimiter(9.0, 18) // 9 msg/sec, burst 18

func signData(input string) xswd.RPCResponse {

	// fmt.Println(estimate)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "SignData",
		"method":  "SignData",
		"params":  []byte(input),
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return xswd.RPCResponse{}
	}

	err = limiter.Wait(context.Background())
	if err != nil {
		panic(err)
	}

	err = xswd_conn.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		panic(err)
	}

	_, msg, err := xswd_conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	var r xswd.RPCResponse
	if err := json.Unmarshal(msg, &r); err != nil {
		return xswd.RPCResponse{}
	}

	return r
}
