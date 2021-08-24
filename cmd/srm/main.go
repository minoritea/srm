package main

import (
	"bytes"
	"flag"
	"github.com/minoritea/srm"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	name := flag.String("type", "", "specify model struct")
	flag.Parse()
	file := os.Getenv("GOFILE")
	if file == "" {
		log.Panic("GOFILE must be given")
	}

	if !strings.HasSuffix(file, ".go") {
		log.Panic(".go file must be given")
	}

	parser := srm.NewParser()

	err := parser.ParseFile(file, []string{*name}).Err()
	if err != nil {
		log.Panicf("%v", err)
	}

	var buf bytes.Buffer
	err = srm.Template.Execute(&buf, parser.Result())
	if err != nil {
		log.Panicf("%v", err)
	}

	newFile := strings.TrimSuffix(file, ".go") + "_srm_generated.go"
	if err != nil {
		log.Panicf("%v", err)
	}
	err = ioutil.WriteFile(newFile, buf.Bytes(), 0666)
	if err != nil {
		log.Panicf("%v", err)
	}
}
