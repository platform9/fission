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

//!+input
const input = `package main

type A struct{}
func (*A) f()

type B int
func (B) f()
func (*B) g()

type I interface { f() }
type J interface { g() }
`

//!-input

func main() {
	// Parse one file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input.go", input, 0)
	if err != nil {
		log.Fatal(err) // parse error
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("hello", fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err) // type error
	}

	//!+implements
	// Find all named types at package level.
	var allNamed []*types.Named
	for _, name := range pkg.Scope().Names() {
		if obj, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok {
			allNamed = append(allNamed, obj.Type().(*types.Named))
		}
	}

	// Test assignability of all distinct pairs of
	// named types (T, U) where U is an interface.
	for _, T := range allNamed {
		for _, U := range allNamed {
			if T == U || !types.IsInterface(U) {
				continue
			}
			if types.AssignableTo(T, U) {
				fmt.Printf("%s satisfies %s\n", T, U)
			} else if !types.IsInterface(T) &&
				types.AssignableTo(types.NewPointer(T), U) {
				fmt.Printf("%s satisfies %s\n", types.NewPointer(T), U)
			}
		}
	}
	//!-implements
}

/*
//!+output
$ go build github.com/golang/example/gotypes/implements
$ ./implements
*hello.A satisfies hello.I
hello.B satisfies hello.I
*hello.B satisfies hello.J
//!-output
*/
