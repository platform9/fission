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
}
`

//!+
func PrintDefsUses(fset *token.FileSet, files ...*ast.File) error {
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	_, err := conf.Check("hello", fset, files, info)
	if err != nil {
		return err // type error
	}

	for id, obj := range info.Defs {
		fmt.Printf("%s: %q defines %v\n",
			fset.Position(id.Pos()), id.Name, obj)
	}
	for id, obj := range info.Uses {
		fmt.Printf("%s: %q uses %v\n",
			fset.Position(id.Pos()), id.Name, obj)
	}
	return nil
}

//!-

func main() {
	// Parse one file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, 0)
	if err != nil {
		log.Fatal(err) // parse error
	}
	if err := PrintDefsUses(fset, f); err != nil {
		log.Fatal(err) // type error
	}
}

/*
//!+output
$ go build github.com/golang/example/gotypes/defsuses
$ ./defsuses
hello.go:1:9: "main" defines <nil>
hello.go:5:6: "main" defines func hello.main()
hello.go:6:9: "fmt" uses package fmt
hello.go:6:13: "Println" uses func fmt.Println(a ...interface{}) (n int, err error)
//!-output
*/
