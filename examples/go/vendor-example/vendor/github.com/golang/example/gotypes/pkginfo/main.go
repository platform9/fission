//!+
package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

const hello = `package main

import "fmt"

func main() {
        fmt.Println("Hello, world")
}`

func main() {
	fset := token.NewFileSet()

	// Parse the input string, []byte, or io.Reader,
	// recording position information in fset.
	// ParseFile returns an *ast.File, a syntax tree.
	f, err := parser.ParseFile(fset, "hello.go", hello, 0)
	if err != nil {
		log.Fatal(err) // parse error
	}

	// A Config controls various options of the type checker.
	// The defaults work fine except for one setting:
	// we must specify how to deal with imports.
	conf := types.Config{Importer: importer.Default()}

	// Type-check the package containing only file f.
	// Check returns a *types.Package.
	pkg, err := conf.Check("cmd/hello", fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err) // type error
	}

	fmt.Printf("Package  %q\n", pkg.Path())
	fmt.Printf("Name:    %s\n", pkg.Name())
	fmt.Printf("Imports: %s\n", pkg.Imports())
	fmt.Printf("Scope:   %s\n", pkg.Scope())
}

//!-

/*
//!+output
$ go build github.com/golang/example/gotypes/pkginfo
$ ./pkginfo
Package  "cmd/hello"
Name:    main
Imports: [package fmt ("fmt")]
Scope:   package "cmd/hello" scope 0x820533590 {
.  func cmd/hello.main()
}
//!-output
*/
