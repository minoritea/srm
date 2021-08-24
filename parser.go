package srm

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"log"
)

type Parser struct {
	pkgName string
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

type Result struct {
	PkgName string
	Models  []Model
}

func (p *Parser) Result() Result {
	var result Result
	result.PkgName = p.pkgName
	for _, model := range p.models {
		result.Models = append(result.Models, model)
	}
	return result
}

func (p *Parser) ParseFile(path string, types []string) *Parser {
	if p.err != nil {
		return p
	}

	f, err := parser.ParseFile(p.FileSet, path, nil, parser.AllErrors)
	if err != nil {
		return p.setError(err)
	}

	p.pkgName = f.Name.Name

	fp := &fileParser{Parser: p, File: f, Dir: filepath.Dir(path)}

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
	Dir string
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

func (fp *fileParser) parseEmbededStruct(ancestors string, f *ast.Field) []field {
	var (
		dirName    string
		structName string
		pkgName    string
	)
	switch t := f.Type.(type) {
	case *ast.Ident:
		if t.Obj != nil {
			ts, ok := t.Obj.Decl.(*ast.TypeSpec)
			if !ok {
				return nil
			}
			return fp.parseStruct(ancestors+"."+ts.Name.Name, ts)
		}

		impPkg, err := build.ImportDir(fp.Dir, 0)
		if err != nil {
			log.Panicf("error=%+v", err)
		}
		structName = t.Name
		pkgName = fp.File.Name.Name
		dirName = impPkg.Dir
		goto PARSE_PACKAGE

	case *ast.SelectorExpr:
		structName = t.Sel.Name
		id, ok := t.X.(*ast.Ident)
		if !ok {
			return nil
		}
		pkgName = id.Name
		for _, imp := range fp.File.Imports {
			unquoted, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				log.Panicf("error=%+v", err)
			}
			impPkg, err := build.Import(unquoted, ".", 0)
			if err != nil {
				log.Panicf("error=%+v", err)
			}
			if impPkg.Name != pkgName {
				continue
			}
			dirName = impPkg.Dir
			goto PARSE_PACKAGE
		}
		return nil
	}

PARSE_PACKAGE:

	pkgs, err := parser.ParseDir(fp.FileSet, dirName, nil, parser.AllErrors)
	if err != nil {
		log.Panicf("error=%+v", err)
	}

	pkg, ok := pkgs[pkgName]
	if !ok || pkg == nil {
		return nil
	}

	for _, file := range pkg.Files {
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
				if ts.Name.Name != structName {
					continue
				}

				newFp := &fileParser{Parser: fp.Parser, File: file, Dir: dirName}
				return newFp.parseStruct(ancestors+"."+structName, ts)
			}
		}
	}
	return nil
}

func (fp *fileParser) parseField(ancestors string, f *ast.Field) []field {
	var result []field
	if len(f.Names) == 0 {
		return fp.parseEmbededStruct(ancestors, f)
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
