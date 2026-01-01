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

var appID = "6df99f80bc8b17340c21fa9c7613e9837cf641b1a1168433e8343337c752073c"
var appSig = `-----BEGIN DERO SIGNED MESSAGE-----
Address: dero1qyc96tgvz8fz623snpfwjgdhlqznamcsuh8rahrh2yvsf2gqqxdljqg9a9kka
C: d30f486cc66f6d6571112fcb3aacba4f076aba439e9bd0e84bef94b06e5c851
S: 2d839f4432e1c7a2da391dd01ed9efec64831b2bbc99a47ab4a04b283005080a

NmRmOTlmODBiYzhiMTczNDBjMjFmYTljNzYxM2U5ODM3Y2Y2NDFiMWExMTY4NDMz
ZTgzNDMzMzdjNzUyMDczYw==
-----END DERO SIGNED MESSAGE-----`
var xswd_conn *websocket.Conn

func set_ws_conn() error {

	websocket_endpoint := "ws://127.0.0.1:44326/xswd"
	var err error

	fmt.Printf("Connecting to %s\n", websocket_endpoint)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // allow self-signed certs
	}

	xswd_conn, _, err = dialer.Dial(websocket_endpoint, nil)
	if err != nil {
		return err
	}
	fmt.Println("WebSocket xswd_connected")
	appData := xswd.ApplicationData{}

	appData.Signature = []byte(appSig)
	appData.Name = "simple-tela-deploymnet-manager"
	appData.Description = "Creating deployments on must be simple and fun! :)"
	appData.Url = "http://localhost:8080"
	appData.Permissions = map[string]xswd.Permission{
		"SignData": xswd.AlwaysAllow, // because that's what this does
	}
	appData.Id = appID
	if err := xswd_conn.WriteJSON(appData); err != nil {
		panic(err)
	}
	fmt.Println("Auth handshake sent")

	_, msg, err := xswd_conn.ReadMessage()
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
