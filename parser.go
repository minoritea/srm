package srm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"log"
)

func ParseFile(file string, names []string) ([]Spec, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, file, nil, parser.AllErrors)
	if err != nil {
		log.Panicf("%+v", err)
	}

	var (
		ts    *ast.TypeSpec
		specs = map[string]Spec{}
	)

	for _, name := range names {
		specs[name] = Spec{
			Name:   name,
			Fields: map[string][]string{},
		}
	}

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

			spec, ok := specs[ts.Name.Name]

			if !ok {
				continue
			}

			spec.scan(nil, ts)
		}
	}

	var result []Spec
	for _, spec := range specs {
		result = append(result, spec)
	}

	return result, nil
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
			name := camelToSnake(n.Name)
			s.Fields[name] = append(
				s.Fields[name],
				strings.Join(append(append([]string{s.Name}, ancestors...), n.Name), "."),
			)
		}
	}
}
