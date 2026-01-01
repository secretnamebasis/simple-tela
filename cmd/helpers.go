package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tela "github.com/secretnamebasis/simple-tela/pkg"
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

func CompileDocs(dURL, base string, contents []string, signed_contents []string) (docs []tela.DOC) {
	for i, each := range contents {

		name := strings.TrimPrefix(each, base)

		// with windows, the trailing slash \ is used by the fs
		// with everyone else, a leading slash / is the case
		// convert to slash element regardless
		name = filepath.ToSlash(name)

		// now remove the first element, the root
		root := 1
		name = name[root:]

		subdir := ""
		// we are going to make an assumption that a file this/file.txt doesn't work
		// and neither does it work to use the trailing slash, \ , eg this\file.txt
		// thus, if the name of the contents contains a leading slash, it is a subdir
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			// remove the final part as it is the base, or filename
			dirs := parts[:(len(parts) - 1)]
			subdir = strings.Join(dirs, "/")
		}

		// fileEx := filepath.Ext(name)

		docType := tela.ParseDocType(name)

		// fmt.Printf("doc-type: %13s file-ext: %7s file-name: %s\n", docType, fileEx, name)

		b, err := base64.StdEncoding.DecodeString(signed_contents[i])
		if err != nil {
			fmt.Println(err)
			return
		}

		result := string(b)

		sig_parts := strings.Split(result, "\n")

		// begin := sig_parts[0]
		address := sig_parts[1]
		c_value := sig_parts[2]
		s_value := sig_parts[3]
		empty := sig_parts[4]
		// end := sig_parts[5]

		// the message is blank when the signature is empty
		message := empty
		if result != "" {
			// end = sig_parts[6]
			message = sig_parts[5]
		}

		// fmt.Println(begin, address, c_value, s_value,
		// 	empty,
		// 	message,
		// 	end,
		// )

		// I guess we could make a table to input all the data
		doc := tela.DOC{
			Code:    message, // this is the contents of the file
			DocType: docType,
			DURL:    dURL, // this is tricky because this is a name-space thing...
			// in tela, .shards is a valid tld

			SubDir: subdir, // this is tricky as well because it is a routing thing
			// Compression: , // this really isn't all that tricky, are we compressing the data?
			// what makes it tricky is the concept of sharding...
			Headers: tela.Headers{
				NameHdr: name, // On-chain name of SC. For TELA-DOCs, they are recreated using this as the file name, it should include the file extension
				// DescrHdr: , // unfortunately, how are we supposed to get this?
				// IconHdr: , // and what about this one... how are we supposed to handle this?
			},
			Signature: tela.Signature{
				CheckC: c_value,
				CheckS: s_value,
			},
			SCVersion: &tela.GetContractVersions(true)[0],
			Author:    address,
		}

		docs = append(docs, doc)
	}
	return
}
