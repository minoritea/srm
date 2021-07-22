package main

import (
	"flag"
	"github.com/minoritea/srm"
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

	newFile := strings.TrimSuffix(file, ".go") + "_srm_generated.go"

	output, err := os.OpenFile(newFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Panicf("%v", err)
	}

	specs, err := srm.ParseFile(file, []string{*name})

	err = srm.Template.Execute(output, specs)
	if err != nil {
		log.Panicf("%v", err)
	}
}
