package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"

	"log"
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

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, os.Getenv("GOFILE"), nil, parser.AllErrors)
	if err != nil {
		log.Panicf("%+v", err)
	}
	var (
		ts    *ast.TypeSpec
		found bool
		spec  = Spec{
			Name:   *name,
			Fields: map[string][]string{},
		}
	)
LOOP:
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gen.Specs {
			var ok bool
			ts, ok = sp.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if ts.Name.Name != spec.Name {
				continue
			}

			spec.scan(nil, ts)

			found = true
			break LOOP
		}
	}

	if !found {
		log.Panic("TypeSpec not found")
	}

	output, err := os.OpenFile(newFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Panicf("%v", err)
	}

	err = tmpl.Execute(output, spec)
	if err != nil {
		log.Panicf("%v", err)
	}
}

type Spec struct {
	Name   string
	Fields map[string][]string
}

func (s *Spec) scan(ancestors []string, ts *ast.TypeSpec) {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}

	if st.Fields == nil {
		return
	}

	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			id, ok := f.Type.(*ast.Ident)
			if !ok {
				continue
			}
			if id.Obj == nil {
				continue
			}
			cts, ok := id.Obj.Decl.(*ast.TypeSpec)
			if !ok {
				continue
			}
			s.scan([]string{id.Obj.Name}, cts)
		}
		for _, n := range f.Names {
			if !isPublic(n.Name) {
				continue
			}
			name := strings.ToLower(n.Name)
			s.Fields[name] = append(
				s.Fields[name],
				strings.Join(append(append([]string{s.Name}, ancestors...), n.Name), "."),
			)
		}
	}
}

var upperCaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // No support for unicode upper case letters.
func isPublic(name string) bool {
	return strings.ContainsAny(name[0:1], upperCaseLetters)
}

var tmpl = template.Must(template.New("*SRM").Parse(`package testdata

import (
	"database/sql"
)

type {{ .Name }}SRMRow struct {
	{{ .Name }}
}

func (row *{{ .Name }}SRMRow) bind(rows *sql.Rows, columns []string) error {
	var (
		dest []interface{}
		{{- range $name, $_ := .Fields }}
		counterOf{{ $name }} int
		{{- end }}
	)
	for _, name := range columns {
		switch name {
		{{- range $name, $fields := .Fields }}
		case "{{ $name }}":
			switch counterOf{{ $name }} {
			{{- range $index, $field := $fields }}
			case {{ $index }}:
				dest = append(dest, &row.{{ $field }})
				counterOf{{ $name }}++
				continue
			{{- end }}
			}
			counterOf{{ $name }}++
		{{- end }}
		}
		var i interface{}
		dest = append(dest, &i)
	}
	return rows.Scan(dest...)
}

type {{ .Name }}SRM []{{ .Name }}SRMRow
func (srm *{{ .Name }}SRM) Bind(rows *sql.Rows, err error) error {
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		var srmRow {{ .Name }}SRMRow
		err := srmRow.bind(rows, columns)
		if err != nil {
			return err
		}
		*srm = append(*srm, srmRow)
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}
`))
