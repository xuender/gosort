package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mvdan.cc/gofumpt/format"
)

var exitCode = 0

// Sort 排序.
func Sort(filename string, src any) ([]byte, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	// ast.Inspect(file, func(n ast.Node) bool {
	// 	return true
	// })
	// astutil.Apply(file, func(c *astutil.Cursor) bool { return true }, func(c *astutil.Cursor) bool { return true })

	// for _, decl := range file.Decls {
	// 	fmt.Println("decl", decl.Pos(), decl.End())
	// }

	// for _, comment := range file.Comments {
	// 	fmt.Println(comment.Pos(), comment.End())
	// }
	// astutil.Cursor
	sort.Slice(file.Decls, func(indexA, indexB int) bool {
		switch typeA := file.Decls[indexA].(type) {
		case *ast.FuncDecl:
			if typeB, ok := file.Decls[indexB].(*ast.FuncDecl); ok {
				if typeA.Name.Name == "main" {
					return true
				}

				if typeB.Name.Name == "main" {
					return false
				}

				return typeA.Name.Name < typeB.Name.Name
			}

			return false
		case *ast.GenDecl:
			if typeB, ok := file.Decls[indexB].(*ast.GenDecl); ok {
				if typeA.Tok == typeB.Tok {
					return compare(typeA.Specs, typeB.Specs)
				}

				return typeA.Tok < typeB.Tok
			}

			return true
		}

		return false
	})

	opts := format.Options{}
	format.File(fset, file, opts)

	// for _, decl := range file.Decls {
	// 	fmt.Println("decl", decl.Pos(), decl.End())
	// }

	// for _, comment := range file.Comments {
	// 	fmt.Println("comment", comment.Pos(), comment.End())
	// }

	printConfig := &printer.Config{Mode: printer.TabIndent, Tabwidth: 4}

	var buf bytes.Buffer

	if err := printConfig.Fprint(&buf, fset, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compare(specsA []ast.Spec, specsB []ast.Spec) bool {
	if len(specsA) == 0 || len(specsB) == 0 {
		return true
	}

	switch typeA := specsA[0].(type) {
	case *ast.ValueSpec:
		typeB, _ := specsB[0].(*ast.ValueSpec)

		return typeA.Names[0].Name < typeB.Names[0].Name
	case *ast.TypeSpec:
		typeB, _ := specsB[0].(*ast.TypeSpec)

		return typeA.Name.Name < typeB.Name.Name
	}

	return false
}

func isGoFile(f os.FileInfo) bool {
	name := f.Name()

	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		if err := processFile("main.go", nil, os.Stdout); err != nil {
			report(err)
		}
	}

	for _, path := range flag.Args() {
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			walkDir(path)
		default:
			if err := processFile(path, nil, os.Stdout); err != nil {
				report(err)
			}
		}
	}

	os.Exit(exitCode)
}

func processFile(filename string, reader io.Reader, writer io.Writer) error {
	if reader == nil {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}

		defer file.Close()

		reader = file
	}

	src, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	target := filename

	res, err := Sort(target, src)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		_, err = writer.Write(res)
	}

	return err
}

func report(err error) {
	scanner.PrintError(os.Stderr, err)

	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "gosort\n\n")
	fmt.Fprintf(os.Stderr, "TODO.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s source.go\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func visitFile(path string, f os.FileInfo, err error) error {
	if err == nil && isGoFile(f) {
		err = processFile(path, nil, os.Stdout)
	}

	if err != nil {
		report(err)
	}

	return nil
}

func walkDir(path string) {
	_ = filepath.Walk(path, visitFile)
}
