package core

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestExportedIdentifiersAreDocumented fails for every exported identifier in
// the module that has no doc comment.
//
// What counts as documented:
//   - types, functions and methods (on exported types): a doc comment;
//   - struct fields and interface methods: a doc comment above or a trailing
//     comment;
//   - constants and variables: a comment above or beside the spec, or a doc
//     comment on the group they are declared in.
//
// Files of every build tag are checked; test files are skipped.
func TestExportedIdentifiersAreDocumented(t *testing.T) {
	var missing []string
	fset := token.NewFileSet()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkgs, err := parser.ParseDir(fset, path, func(fi fs.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				missing = append(missing, undocumented(fset, file)...)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("no doc comment: %s", m)
	}
}

// undocumented returns "pkg.Name (file:line)" for each exported identifier of
// file that lacks a doc comment.
func undocumented(fset *token.FileSet, file *ast.File) []string {
	var out []string
	report := func(pos token.Pos, name string) {
		p := fset.Position(pos)
		out = append(out, fmt.Sprintf("%s.%s (%s:%d)", file.Name.Name, name, filepath.ToSlash(p.Filename), p.Line))
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() || d.Doc != nil {
				continue
			}
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := receiverName(d.Recv.List[0].Type)
				if !ast.IsExported(recv) {
					continue
				}
				name = recv + "." + name
			}
			report(d.Pos(), name)

		case *ast.GenDecl:
			groupDoc := d.Doc != nil
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if !s.Name.IsExported() {
						continue
					}
					if s.Doc == nil && !groupDoc {
						report(s.Pos(), s.Name.Name)
					}
					out = append(out, undocumentedMembers(fset, file, s)...)
				case *ast.ValueSpec:
					if s.Doc != nil || s.Comment != nil || groupDoc {
						continue
					}
					for _, n := range s.Names {
						if n.IsExported() {
							report(n.Pos(), n.Name)
						}
					}
				}
			}
		}
	}
	return out
}

// undocumentedMembers checks the exported fields of a struct type and the
// exported methods of an interface type.
func undocumentedMembers(fset *token.FileSet, file *ast.File, s *ast.TypeSpec) []string {
	var fields *ast.FieldList
	switch t := s.Type.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	default:
		return nil
	}

	var out []string
	for _, f := range fields.List {
		if f.Doc != nil || f.Comment != nil {
			continue
		}
		for _, n := range f.Names { // embedded fields have no names and are skipped
			if n.IsExported() {
				p := fset.Position(n.Pos())
				out = append(out, fmt.Sprintf("%s.%s.%s (%s:%d)", file.Name.Name, s.Name.Name, n.Name, filepath.ToSlash(p.Filename), p.Line))
			}
		}
	}
	return out
}

// receiverName returns the type name of a method receiver, without pointer
// and type parameters.
func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
