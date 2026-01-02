package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/secretnamebasis/simple-tela/cmd"
	tela "github.com/secretnamebasis/simple-tela/pkg"
)

var network string = "simulator"
var signature string //= rand.Text()
var default_simulator_port = "20000"
var default_mainnet_port = "10102"
var docs []tela.DOC
var headers = []string{
	"DocType",
	"Code", // I don't think we can show this...
	"SubDir",
	"DURL",
	"NameHdr",
	"DescHdr",
	"IconHdr",
}

func RenderGui() {
	a := app.NewWithID("simple-tela" + rand.Text())
	w := a.NewWindow("simple-tela")
	w.Resize(fyne.NewSize(1200, 300))
	var table *widget.Table
	length := func() (rows int, cols int) {
		return len(docs), len(headers)
	}

	create := func() fyne.CanvasObject {
		e := widget.NewEntry()
		e.Hide()
		e.Disable()

		l := widget.NewHyperlink("", nil)
		l.Hide()

		return container.NewStack(e, l)
	}

	update := func(id widget.TableCellID, co fyne.CanvasObject) {
		if id.Row < 0 || id.Row >= len(docs) {
			return
		}

		template := co.(*fyne.Container)

		// capture components
		entry := template.Objects[0].(*widget.Entry)
		link := template.Objects[1].(*widget.Hyperlink)
		row := id.Row

		// model updaters
		updateDescrHdr := func(s string) {
			docs[row].DescrHdr = s
		}
		updateIconHdr := func(s string) {
			docs[row].IconHdr = s
		}

		codeDialog := func() {
			w := a.NewWindow("code-viewer")
			content := container.NewBorder(nil, nil, nil, nil, container.NewScroll(widget.NewLabel(docs[row].Code)))
			w.Resize(fyne.NewSize(800, 300))
			w.SetContent(content)
			w.Show()
		}

		// Reset reused widgets
		entry.Disable()
		entry.OnChanged = nil
		entry.SetText("")
		entry.Hide()

		link.OnTapped = nil
		link.SetText("")
		link.Hide()

		switch id.Col {
		case 0:
			entry.Show()
			entry.SetText(docs[row].DocType)

		case 1:
			link.Show()
			link.SetText("view code")
			link.OnTapped = codeDialog

		case 2:
			entry.Show()
			entry.SetText(docs[row].SubDir)

		case 3:
			entry.Show()
			entry.SetText(docs[row].DURL)

		case 4:
			entry.Show()
			entry.SetText(docs[row].NameHdr)

		case 5:
			entry.Show()
			entry.Enable()
			entry.SetText(docs[row].DescrHdr)
			entry.OnChanged = updateDescrHdr

		case 6:
			entry.Show()
			entry.Enable()
			entry.SetText(docs[row].IconHdr)
			entry.OnChanged = updateIconHdr
		}
	}

	table = widget.NewTable(length, create, update)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("") }
	table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		label := template.(*widget.Label)
		if id.Row < 0 {
			label.SetText(headers[id.Col])
			return
		}

	}
	for i := range len(headers) {
		table.SetColumnWidth(i, 150)
	}
	dUrl := widget.NewEntry()
	dUrl.SetPlaceHolder("DURL of deployment")
	dUrl.Validator = func(s string) error {
		if s == "" {
			return errors.New("empty durl")
		}
		return nil
	}
	table_contents := []string{}
	src := widget.NewEntry()
	src.SetPlaceHolder("source file for deployment")
	src.OnChanged = func(s string) {
		contents := []string{}
		paths, err := os.ReadDir(s)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) && !strings.Contains(s, ".json") {
				fmt.Println(err)
				fmt.Printf("err type: %T\n", err)
			}
			return
		}

		cmd.Walk(s, paths, &contents)
		// now we need to select what we want to deploy...
		table_contents = contents
	}
	upload_folder := func() {
		if dUrl.Text == "" {
			dUrl.SetText(" ")
			dUrl.SetText("")
		}
		if err := dUrl.Validate(); err != nil {
			return
		}
		fo := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil {
				fmt.Println(err)
				return
			}
			if lu == nil {
				return
			}
			src.SetText(lu.Path())
		}, w)
		fo.Resize(fyne.NewSize(800, 300))
		fo.Show()
	}
	compile := func() {
		if dUrl.Text == "" {
			dUrl.SetText(" ")
			dUrl.SetText("")
			return
		}
		if src.Text == "" {
			return
		}
		fileBytes, readErr := os.ReadFile(src.Text)
		if readErr == nil {
			if err := json.Unmarshal(fileBytes, &docs); err != nil {
				fmt.Println(err)
				return
			}
			jsonBytes, err := json.MarshalIndent(docs, "", " ")
			if err != nil {
				fmt.Println(err)
				return
			}
			os.WriteFile(filepath.Join("src", "docs.json"), jsonBytes, 0644)

			table.Refresh()
			return
		} else {
			docs = []tela.DOC{}

			fmt.Println("asking for endpoint at xswd websocket connection")

			if err := set_ws_conn(); err != nil {
				dialog.ShowError(err, w)
				fmt.Println(err)
				return
			}

			d := getDaemonEndpoint()

			if d.Endpoint == "" {
				err := errors.New("endpoint cannot be empty")
				dialog.ShowError(err, w)
				fmt.Println(err)
				return
			}

			fmt.Println("endpoint", d.Endpoint)

			u, err := url.Parse("http://" + d.Endpoint)
			if err != nil {
				dialog.ShowError(err, w)
				fmt.Println(err)
				return
			}
			// parts := strings.Split(r.Endpoint, ":")

			switch u.Port() {
			case default_simulator_port:
				network = "simulator"
			case default_mainnet_port:
				fallthrough
			default:
				network = "mainnet"
			}

			fmt.Println("attaching to " + network + " , daemon is on port: " + u.Port())

			// there are two parts to the deployment process:

			// building docs & mods,
			// var mods []tela.MOD

			// // and then building the index
			// var index tela.INDEX

			fmt.Println("parsing contents of", src.Text)
			signed_docs := []string{}
			for _, each := range table_contents {

				fileBytes, err := os.ReadFile(each)
				if err != nil {
					fmt.Println(err)
					continue
				}
				code := string(fileBytes)
				r := signData(code) // it would be fun to put data in here for creating a signature

				data := r.Result.(map[string]any)["signature"].(string)
				signed_docs = append(signed_docs, data)
			}

			docs = cmd.CompileDocs(dUrl.Text, src.Text, table_contents, signed_docs)
			// // probably fun to integrate the concept of the appDataID here...
			// // I think the deciding factor will be to determine if there is going to be an ws connect
			// // we should parse the application data that we get and then find out...
			// results.SetText(strings.Join(contents, "\n"))

			xswd_conn.Close()
		}

		table.Refresh()
	}
	src.ActionItem = widget.NewButtonWithIcon("compile docs", theme.FileIcon(), compile)
	src.Disable()

	install := func() {
		if dUrl.Text == "" {
			dUrl.SetText(" ")
			dUrl.SetText("")
		}
		if err := dUrl.Validate(); err != nil {
			fmt.Println(err)
			return
		}
		jsonBytes, err := json.MarshalIndent(docs, "", " ")
		if err != nil {
			fmt.Println(err)
			return
		}
		os.WriteFile(filepath.Join("src", "docs.json"), jsonBytes, 0644)

		// byts, err := json.Marshal(docs)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
		os.Args = append(os.Args, ("--durl=" + dUrl.Text))
		os.Args = append(os.Args, (`--src-file=` + filepath.Join("src", "docs.json")))
		// os.Args = append(os.Args, ("--src-json=" + string(byts))) // the bytes aren't saved to a deployment

		cmd.Run()
	}
	src.OnSubmitted = func(s string) { compile() }
	choose_folder := widget.NewButton("choose folder", upload_folder) // at this point, when we hit install, we are validating the docs
	importDocs := func() {
		if dUrl.Text == "" {
			dUrl.SetText(" ")
			dUrl.SetText("")
			return
		}
		fo := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				fmt.Println(err)
				return
			}
			if reader == nil {
				return
			}
			src.SetText(reader.URI().Path())
			if src.Text == "" {
				return
			}
		}, w)
		fo.Resize(fyne.NewSize(800, 300))
		fo.Show()
	}
	import_docs := widget.NewButton("or, import docs.json", importDocs)
	install_docs := widget.NewButton("install docs", install)
	content := container.NewBorder(container.NewVBox(container.NewVBox(dUrl, src), container.NewAdaptiveGrid(3, choose_folder, import_docs, install_docs)), nil, nil, nil, table)
	w.SetContent(content)
	w.ShowAndRun()
}
