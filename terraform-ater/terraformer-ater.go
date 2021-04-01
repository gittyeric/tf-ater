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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type node struct {
	id         string
	rname      string
	outputForm string
	tfForm     string
	uses       map[string]bool
	usedBy     map[string]bool
	data       *hclwrite.Block
}

func newNode(id string, name string, data *hclwrite.Block) *node {
	n := node{data: data,
		outputForm: id + "_" + name,
		tfForm:     id + "." + name,
		id:         id,
		rname:      name}

	n.usedBy = make(map[string]bool)
	n.uses = make(map[string]bool)
	return &n
}

func parseCommandLine() (string, string, string, string, error) {
	if len(os.Args) != 6 {
		fmt.Println("Usage: terraformer-output-dir tf-referenceType target-refference-name outputFileWeWant.tf ssl_certificates.tf")
		fmt.Println("Example: " + os.Args[0] + " generated/google google_compute_instance foo-bar-disp-name compute_inst.tf")
		return "", "", "", "", fmt.Errorf("invalid command argument")
	}

	return os.Args[1], os.Args[2] + "_" + os.Args[3], os.Args[4], os.Args[5], nil
}

func getCertVals(sslCertFile string) (*map[string]string, error) {
	sslTfFile, err := ioutil.ReadFile(sslCertFile)
	if err != nil {
		fmt.Printf("Error accessing %q: %v\n", sslCertFile, err)
		return nil, err
	}
	file, diag := hclwrite.ParseConfig(sslTfFile, "", hcl.Pos{Line: 1, Column: 1})
	if diag.HasErrors() {
		fmt.Println(diag)
		return nil, fmt.Errorf("invalid hcl")
	}
	certsMap := make(map[string]string)
	for _, block := range file.Body().Blocks() {
		if block.Type() != "data" {
			continue
		}

		labels := block.Labels()
		if len(labels) != 2 {
			continue
		}

		for key, attr := range block.Body().Attributes() {
			if key != "name" {
				continue
			}
			var tokens hclwrite.Tokens
			line := string(attr.BuildTokens(tokens).Bytes())
			captureNameSslValues := regexp.MustCompile(`name[ ]+=[ ]+"(.+)"`)
			match := captureNameSslValues.FindStringSubmatch(line)
			if len(match) != 2 {
				continue
			}
			certsMap[match[1]] = labels[1]
		}
	}
	return &certsMap, nil
}

func getResourceKey(block *hclwrite.Block) (string, string, string, error) {
	if block.Type() != "resource" {
		return "", "", "", fmt.Errorf("missing resource")
	}

	labels := block.Labels()
	if len(labels) != 2 {
		return "", "", "", fmt.Errorf("invalid resource labels")
	}

	key := labels[0] + "_" + labels[1]
	return key, labels[0], labels[1], nil
}

func parseBlocks(file *hclwrite.File, resourceGraph *map[string]*node, override bool) {
	graph := *resourceGraph
	for _, block := range file.Body().Blocks() {
		key, id, rname, err := getResourceKey(block)
		if err != nil {
			continue
		}
		if _, ok := graph[key]; ok && !override {
			continue
		}

		labelsList := block.Labels()
		dashSwap := labelsList[1]
		noDashes := strings.Replace(dashSwap, "-", "_", -1)
		labelsList[1] = noDashes

		block.SetLabels(labelsList)
		graph[key] = newNode(id, rname, block)
	}
}

func initializeGraph(tfdir string) *map[string]*node {
	resourceGraph := make(map[string]*node)
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
		}
		file, diag := hclwrite.ParseConfig(tfFile, "", hcl.Pos{Line: 1, Column: 1})
		if diag.HasErrors() {
			fmt.Println(diag)
			return error(nil)
		}
		if strings.Contains(path, "/global/") {
			parseBlocks(file, &resourceGraph, true)
		} else {
			parseBlocks(file, &resourceGraph, false)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return &resourceGraph
}

func parseLines(resourceGraph *map[string]*node, node *node, certsMap *map[string]string) {
	graph := *resourceGraph
	certs := *certsMap
	for key, attr := range node.data.Body().Attributes() {
		var tokens hclwrite.Tokens
		line := string(attr.BuildTokens(tokens).Bytes())
		captureAssignmentValues := regexp.MustCompile(`(.*)[ ]+=[ ]+\[?"\${data.*\.outputs\.(.*)}"(\])?\n$`)
		match := captureAssignmentValues.FindStringSubmatch(line)
		if len(match) == 4 {
			captureAssignmentValues = regexp.MustCompile(`(.*)[._]self_links`)
			rkey := captureAssignmentValues.FindStringSubmatch(match[2])
			if len(rkey) == 2 {
				graphKey := rkey[1]
				if val, ok := graph[graphKey]; ok {
					val.usedBy[node.outputForm] = true
					if match[3] == "" {
						node.data.Body().SetAttributeTraversal(
							key,
							hcl.Traversal{hcl.TraverseRoot{Name: val.id + "." + val.rname + ".self_link"}})
					} else {
						node.data.Body().SetAttributeTraversal(
							key,
							hcl.Traversal{hcl.TraverseRoot{Name: "" + val.id + "." + val.rname + ".self_link"}})
					}
					node.uses[graphKey] = true
				}
			}
		}
		if key == "ssl_certificates" {
			captureSslCertValues := regexp.MustCompile(`.*[ ]+=[ ]+\["https.*sslCertificates/(.*)"]`)
			match = captureSslCertValues.FindStringSubmatch(line)
			node.data.Body().SetAttributeTraversal(
				key,
				hcl.Traversal{hcl.TraverseRoot{Name: "[data.google_compute_ssl_certificate." + certs[match[1]] + ".self_link]"}})
		}
	}
}

func appendToBody(body *hclwrite.Body, resourceGraph *map[string]*node, node *node, firstNode bool, visitedSet *map[string]bool) {
	graph := *resourceGraph
	visited := *visitedSet
	if len(node.usedBy) > 1 && !firstNode {
		fmt.Printf("Skipping %s %s", node.id, node.rname)
		return
	} else if len(node.usedBy) < 2 {
		for k := range node.usedBy {
			fmt.Printf("Used by %s %s\n", graph[k].id, graph[k].rname)
			if _, ok := visited[k]; ok {
				continue
			}
			fmt.Printf("Adding Usedby %s %s\n", graph[k].id, graph[k].rname)
			visited[k] = true
			appendToBody(body, resourceGraph, graph[k], false, visitedSet)
		}
	}
	for k := range node.uses {
		fmt.Printf("Use %s %s\n", graph[k].id, graph[k].rname)
		if _, ok := visited[k]; ok {
			continue
		}
		fmt.Printf("Adding %s %s\n", graph[k].id, graph[k].rname)
		visited[k] = true
		appendToBody(body, resourceGraph, graph[k], false, visitedSet)
	}
	fmt.Printf("added %s %s\n", node.id, node.rname)
	body.AppendBlock(node.data)
}

func createNewTf(rkey string, resourceGraph *map[string]*node) (*hclwrite.File, error) {
	graph := *resourceGraph
	node, ok := graph[rkey]
	if !ok {
		fmt.Println("Key not found in tf data")
		return nil, fmt.Errorf("requested key not found")
	}
	file := hclwrite.NewFile()
	body := file.Body()
	visited := make(map[string]bool)
	visited[node.outputForm] = true
	appendToBody(body, resourceGraph, node, true, &visited)
	return file, nil
}

func main() {
	tdir, rkey, outputFname, sslCertFile, err := parseCommandLine()
	if err != nil {
		return
	}

	certsMap, err := getCertVals(sslCertFile)
	if err != nil {
		return
	}

	resourceGraph := initializeGraph(tdir)
	if resourceGraph == nil {
		return
	}

	for _, value := range *resourceGraph {
		parseLines(resourceGraph, value, certsMap)
	}

	outfile, err := createNewTf(rkey, resourceGraph)
	if err != nil {
		return
	}
	ioutil.WriteFile(outputFname, outfile.Bytes(), 0644)
}
