package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"strings"
)

//!+input
const hello = `
package main

import "fmt"

// append
func main() {
        // fmt
        fmt.Println("Hello, world")
        // main
        main, x := 1, 2
        // main
        print(main, x)
        // x
}
// x
`

//!-input

//!+main
func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", hello, parser.ParseComments)
	if err != nil {
		log.Fatal(err) // parse error
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("cmd/hello", fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err) // type error
	}

	// Each comment contains a name.
	// Look up that name in the innermost scope enclosing the comment.
	for _, comment := range f.Comments {
		pos := comment.Pos()
		name := strings.TrimSpace(comment.Text())
		fmt.Printf("At %s,\t%q = ", fset.Position(pos), name)
		inner := pkg.Scope().Innermost(pos)
		if _, obj := inner.LookupParent(name, pos); obj != nil {
			fmt.Println(obj)
		} else {
			fmt.Println("not found")
		}
	}
}

//!-main

/*
//!+output
$ go build github.com/golang/example/gotypes/lookup
$ ./lookup
At hello.go:6:1,        "append" = builtin append
At hello.go:8:9,        "fmt" = package fmt
At hello.go:10:9,       "main" = func cmd/hello.main()
At hello.go:12:9,       "main" = var main int
At hello.go:14:9,       "x" = var x int
At hello.go:16:1,       "x" = not found
//!-output
*/
