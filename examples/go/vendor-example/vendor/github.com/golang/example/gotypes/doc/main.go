// The doc command prints the doc comment of a package-level object.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"log"
	"os"

	// TODO: these will use std go/types after Feb 2016
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types/typeutil"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: doc <package> <object>")
	}
	//!+part1
	pkgpath, name := os.Args[1], os.Args[2]

	// The loader loads a complete Go program from source code.
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Import(pkgpath)
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err) // load error
	}

	// Find the package and package-level object.
	pkg := lprog.Package(pkgpath).Pkg
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		log.Fatalf("%s.%s not found", pkg.Path(), name)
	}
	//!-part1
	//!+part2

	// Print the object and its methods (incl. location of definition).
	fmt.Println(obj)
	for _, sel := range typeutil.IntuitiveMethodSet(obj.Type(), nil) {
		fmt.Printf("%s: %s\n", lprog.Fset.Position(sel.Obj().Pos()), sel)
	}

	// Find the path from the root of the AST to the object's position.
	// Walk up to the enclosing ast.Decl for the doc comment.
	_, path, _ := lprog.PathEnclosingInterval(obj.Pos(), obj.Pos())
	for _, n := range path {
		switch n := n.(type) {
		case *ast.GenDecl:
			fmt.Println("\n", n.Doc.Text())
			return
		case *ast.FuncDecl:
			fmt.Println("\n", n.Doc.Text())
			return
		}
	}
	//!-part2
}

/*
//!+output
$ ./doc net/http File
type net/http.File interface{Readdir(count int) ([]os.FileInfo, error); Seek(offset int64, whence int) (int64, error); Stat() (os.FileInfo, error); io.Closer; io.Reader}
/go/src/io/io.go:92:2: method (net/http.File) Close() error
/go/src/io/io.go:71:2: method (net/http.File) Read(p []byte) (n int, err error)
/go/src/net/http/fs.go:65:2: method (net/http.File) Readdir(count int) ([]os.FileInfo, error)
/go/src/net/http/fs.go:66:2: method (net/http.File) Seek(offset int64, whence int) (int64, error)
/go/src/net/http/fs.go:67:2: method (net/http.File) Stat() (os.FileInfo, error)

 A File is returned by a FileSystem's Open method and can be
served by the FileServer implementation.

The methods should behave the same as those on an *os.File.
//!-output
*/
