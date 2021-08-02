package srm

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"log"
)

type Parser struct {
	*token.FileSet
	models map[string]Model
	err    error
}

func NewParser() *Parser {
	return &Parser{
		FileSet: token.NewFileSet(),
		models:  map[string]Model{},
	}
}

func (p *Parser) setError(err error) *Parser {
	p.err = err
	return p
}

func (p *Parser) Err() error { return p.err }

func (p *Parser) Models() []Model {
	var result []Model
	for _, model := range p.models {
		result = append(result, model)
	}
	return result
}

func (p *Parser) ParseFile(file string, types []string) *Parser {
	if p.err != nil {
		return p
	}

	f, err := parser.ParseFile(p.FileSet, file, nil, parser.AllErrors)
	if err != nil {
		return p.setError(err)
	}

	fp := &fileParser{Parser: p, File: f}

	names := map[string]struct{}{}
	for _, name := range types {
		names[name] = struct{}{}

	}

	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gen.Specs {
			var ok bool
			ts, ok := sp.(*ast.TypeSpec)
			if !ok {
				continue
			}

			_, ok = names[ts.Name.Name]
			if !ok {
				continue
			}

			model := fp.parseModel(ts)
			if len(model.Fields) == 0 {
				continue
			}
			p.models[ts.Name.Name] = model
		}
	}

	return p
}

type fileParser struct {
	*Parser
	*ast.File
}

type Model struct {
	Name   string
	Fields map[string][]string
}

func (fp *fileParser) parseModel(ts *ast.TypeSpec) Model {
	name := ts.Name.Name
	model := Model{Name: name, Fields: map[string][]string{}}
	for _, fld := range fp.parseStruct("", ts) {
		model.Fields[fld.columnName] = append(model.Fields[fld.columnName], fld.completeName)
	}
	return model
}

type field struct {
	columnName, completeName string
}

func (fp *fileParser) parseEmbededStruct(ancestors string, ident *ast.Ident) []field {
	if ident.Obj == nil {
		return nil
	}
	ts, ok := ident.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return nil
	}
	return fp.parseStruct(ancestors+"."+ts.Name.Name, ts)
}

func (fp *fileParser) parseStruct(ancestors string, ts *ast.TypeSpec) []field {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	if st.Fields == nil {
		return nil
	}

	var result []field
	for _, f := range st.Fields.List {
		result = append(result, fp.parseField(ancestors, f)...)
	}
	return result
}

func (fp *fileParser) parseExternalStruct(ancestors string, sl *ast.SelectorExpr) []field {
	id, ok := sl.X.(*ast.Ident)
	if !ok {
		return nil
	}
	for _, imp := range fp.File.Imports {
		unquoted, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			log.Panicf("error=%+v", err)
		}
		pkg, err := build.Import(unquoted, ".", 0)
		if err != nil {
			log.Panicf("error=%+v", err)
		}
		if pkg.Name != id.Name {
			continue
		}
		pkgs, err := fp.parsePackage(pkg)
		if err != nil {
			log.Panicf("error=%+v", err)
		}
		pk, ok := pkgs[id.Name]
		if !ok {
			continue
		}
		for _, file := range pk.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, sp := range gen.Specs {
					var ok bool
					ts, ok := sp.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Name.Name != sl.Sel.Name {
						continue
					}
					newFp := &fileParser{Parser: fp.Parser, File: file}
					return newFp.parseStruct(ancestors+"."+sl.Sel.Name, ts)
				}
			}
		}
	}
	return nil
}

func (fp *fileParser) parsePackage(pkg *build.Package) (map[string]*ast.Package, error) {
	return parser.ParseDir(fp.FileSet, pkg.Dir, nil, parser.AllErrors)
}

func (fp *fileParser) parseField(ancestors string, f *ast.Field) []field {
	var result []field
	if len(f.Names) == 0 {
		switch t := f.Type.(type) {
		case *ast.Ident:
			return fp.parseEmbededStruct(ancestors, t)
		case *ast.SelectorExpr:
			return fp.parseExternalStruct(ancestors, t)
		}
		return nil
	}

	var tag = fp.parseTag(f.Tag)
	for i, n := range f.Names {
		columnName := camelToSnake(n.Name)
		if i == len(f.Names)-1 {
			if tag.skip {
				continue
			}

			if tag.name != nil {
				columnName = *tag.name
			}
		}

		if !isPublic(n.Name) {
			continue
		}

		result = append(result, field{
			columnName:   columnName,
			completeName: ancestors + "." + n.Name,
		})
	}
	return result
}

type tag struct {
	name *string
	skip bool
}

func (fp *fileParser) parseTag(t *ast.BasicLit) tag {
	if t == nil {
		return tag{}
	}

	unquote, err := strconv.Unquote(t.Value)
	if err != nil {
		log.Printf("warn: failed to unquote: value=%q, error=%+v", t.Value, err)
		return tag{}
	}
	st := reflect.StructTag(unquote)
	tagv, ok := st.Lookup("srm")
	if !ok {
		return tag{}
	}

	var (
		skip bool
		name *string
	)
	for _, v := range strings.Split(tagv, ",") {
		if v == "-" {
			skip = true
			continue
		}

		kv := strings.Split(v, "=")
		if len(kv) == 2 && kv[0] == "name" {
			name = &kv[1]
			continue
		}
	}

	return tag{skip: skip, name: name}
}
