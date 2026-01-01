package cmd_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	tela "github.com/secretnamebasis/simple-tela/pkg"
)

var wallet_1_db = ""

var index_html = ""

func TestXxx(t *testing.T) {
	t.Run(
		"tela start",
		func(t *testing.T) {
			wait := time.Second * 2

			endpoints := make(map[string]string, 1)

			// endpoints["mainnet"] = "192.168.86.21:10102"

			endpoints["simulator"] = "127.0.0.1:20000"

			var (
				env, endpoint string = "simulator", endpoints["simulator"]
				testnet       bool   = true
			)

			if ip_port, ok := endpoints["mainnet"]; ok {
				env = "prod"
				endpoint = ip_port
				testnet = false
			} else {
				globals.Arguments["--simulator"] = true
			}

			globals.Arguments["--testnet"] = testnet
			globals.Arguments["--daemon-address"] = endpoint

			globals.InitNetwork()
			walletapi.Connect(endpoint)
			go walletapi.Keep_Connectivity()

			// wallet 0 is origin wallet, can't send to self error
			// use one of the other 21 wallets
			wallet, err := walletapi.Open_Encrypted_Wallet(wallet_1_db, "")
			if err != nil {
				t.Errorf("err: %s", err)
			}
			wallet.SetNetwork(!globals.Arguments["--testnet"].(bool))
			wallet.SetDaemonAddress(endpoint)
			wallet.SetOnlineMode()

			a, c, s, err := tela.ParseSignature(wallet.SignData([]byte("")))
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("a:", a, "\nc:", c, "\ns:", s)

			var name string = "icon.svg"

			name = "index.html"
			file, err := os.ReadFile(name)
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("loaded into memory:", name)
			html_index := tela.DOC{
				Code:    string(file),
				DocType: tela.DOC_HTML,
				SubDir:  "",
				DURL:    "app_name.doc",
				Headers: tela.Headers{
					NameHdr:  "index.html",
					DescrHdr: "app_name index.html",
					IconHdr:  "",
				},
				Signature: tela.Signature{
					CheckC: c,
					CheckS: s,
				},
			}
			fmt.Println("loaded into tela doc:", name)

			var args rpc.Arguments
			args, err = tela.NewInstallArgs(&html_index)
			if err != nil {
				return
			}
			doc := ""

			// doc, err := tela.Installer(wallet, 2, ) // it return scids
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("INSTALLED", name, args)
			fmt.Println("deploying to:", env)
			time.Sleep(wait)

			file, err = os.ReadFile(name)
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("loaded into memory:", name)
			svg_icon := tela.DOC{
				Code:    string(file),
				DocType: tela.DOC_STATIC,
				SubDir:  "",
				DURL:    "app_name.svg",
				Headers: tela.Headers{
					NameHdr:  "app_name.svg",
					DescrHdr: "app_name app_name.svg",
					IconHdr:  "",
				},
				Signature: tela.Signature{
					CheckC: c,
					CheckS: s,
				},
			}
			fmt.Println("loaded into tela doc:", name)
			svg, err := tela.Installer(wallet, 2, &svg_icon)
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("INSTALLED", name, svg)
			fmt.Println("deploying to:", env)
			time.Sleep(wait)

			tela_index := tela.INDEX{
				DOCs:   []string{doc, svg},
				Author: a,
				DURL:   "app_name.tela",
				Headers: tela.Headers{
					NameHdr:  "App Name",
					DescrHdr: "App Description.",
					IconHdr:  "",
				},
			}
			fmt.Println("loaded into tela index:", doc, svg)
			index, err := tela.Installer(wallet, 2, &tela_index)
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println("INSTALLED", "INDEX: ", index)
			fmt.Println("deploying to:", env)
			time.Sleep(wait)
			// url, err := tela.ServeTELA(index, endpoint)
			tela_link := "tela://open/" + index + "/index.html"
			url, err := tela.OpenTELALink(tela_link, endpoint)
			if err != nil {
				t.Errorf("err: %s", err)
			}
			fmt.Println(url)
			for {
				time.Sleep(wait)
			}
		},
	)
}

// this is a SinglePageApplication, which means that we are going to have to re-think stuff.
// The best tool would be to make it possible to configure the css for a height clock
// This tool would turn on once you started dedicating hashes
// The alternative, of course, is the brownnoise tool that you power on with EPOCH
