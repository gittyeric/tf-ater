/*
MIT License

Copyright (c) 2021

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"),
to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice (including the next paragraph) shall be included in all copies or
	substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR
ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH
THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func parseCommandLine() (string, string, string, string, error) {
	if len(os.Args) != 5 {
		fmt.Println("Usage: terraformer-output-dir tf-referenceType wrapper-outdir ssl_certificates-tf-file")
		fmt.Println("Example: " + os.Args[0] + " tfdir a outdir/ ssl_certs.tf")
		return "", "", "", "", fmt.Errorf("invalid command argument")
	}

	return os.Args[1], os.Args[2], os.Args[3], os.Args[4], nil
}

func parseBlocks(file *hclwrite.File, cmds *map[string]*exec.Cmd, tfdir string, typeID string, outDir string, sslCertFile string, override bool) {
	rnameToCmds := *cmds
	for _, block := range file.Body().Blocks() {
		if block.Type() == "resource" && len(block.Labels()) == 2 && block.Labels()[0] == typeID {
			rname := block.Labels()[1]
			cmd := exec.Command("./terraformer-ater/terraformer-ater", tfdir, typeID, rname, path.Join(outDir, strings.Replace(rname, "-", "_", -1)+".tf"), sslCertFile)
			if _, ok := rnameToCmds[rname]; ok && !override {
				continue
			}
			rnameToCmds[rname] = cmd
		}
	}
}

func runTerraformerator(tfdir string, typeID string, outDir string, sslCertFile string) {
	rnamesToCmds := make(map[string]*exec.Cmd)
	err := filepath.Walk(tfdir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing %q: %v\n", path, err)
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".tf" {
			return nil
		}
		tfFile, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Error accessing %q: %v\n", path, err)
			return nil
		}
		file, diag := hclwrite.ParseConfig(tfFile, "", hcl.Pos{Line: 1, Column: 1})
		if diag.HasErrors() {
			fmt.Println(diag)
			return error(nil)
		}
		if strings.Contains(path, "/global/") {
			parseBlocks(file, &rnamesToCmds, tfdir, typeID, outDir, sslCertFile, true)
		} else {
			parseBlocks(file, &rnamesToCmds, tfdir, typeID, outDir, sslCertFile, false)
		}
		return nil
	})
	if err != nil {
		return
	}
	os.MkdirAll(outDir, 0755)
	for _, cmd := range rnamesToCmds {
		cmd.Run()
	}
}

func main() {
	tfdir, typeID, outDir, sslCertFile, err := parseCommandLine()
	if err != nil {
		return
	}

	runTerraformerator(tfdir, typeID, outDir, sslCertFile)
}
