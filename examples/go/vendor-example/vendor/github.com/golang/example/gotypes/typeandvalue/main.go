package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

//!+input
const input = `
package main

var m = make(map[string]int)

func main() {
	v, ok := m["hello, " + "world"]
	print(rune(v), ok)
}
`

//!-input

var fset = token.NewFileSet()

func main() {
	f, err := parser.ParseFile(fset, "hello.go", input, 0)
	if err != nil {
		log.Fatal(err) // parse error
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	if _, err := conf.Check("cmd/hello", fset, []*ast.File{f}, info); err != nil {
		log.Fatal(err) // type error
	}

	//!+inspect
	// f is a parsed, type-checked *ast.File.
	ast.Inspect(f, func(n ast.Node) bool {
		if expr, ok := n.(ast.Expr); ok {
			if tv, ok := info.Types[expr]; ok {
				fmt.Printf("%-24s\tmode:  %s\n", nodeString(expr), mode(tv))
				fmt.Printf("\t\t\t\ttype:  %v\n", tv.Type)
				if tv.Value != nil {
					fmt.Printf("\t\t\t\tvalue: %v\n", tv.Value)
				}
			}
		}
		return true
	})
	//!-inspect
}

// nodeString formats a syntax tree in the style of gofmt.
func nodeString(n ast.Node) string {
	var buf bytes.Buffer
	format.Node(&buf, fset, n)
	return buf.String()
}

// mode returns a string describing the mode of an expression.
func mode(tv types.TypeAndValue) string {
	s := ""
	if tv.IsVoid() {
		s += ",void"
	}
	if tv.IsType() {
		s += ",type"
	}
	if tv.IsBuiltin() {
		s += ",builtin"
	}
	if tv.IsValue() {
		s += ",value"
	}
	if tv.IsNil() {
		s += ",nil"
	}
	if tv.Addressable() {
		s += ",addressable"
	}
	if tv.Assignable() {
		s += ",assignable"
	}
	if tv.HasOk() {
		s += ",ok"
	}
	return s[1:]
}

/*
//!+output
$ go build github.com/golang/example/gotypes/typeandvalue
$ ./typeandvalue
make(map[string]int)            mode:  value
                                type:  map[string]int
make                            mode:  builtin
                                type:  func(map[string]int) map[string]int
map[string]int                  mode:  type
                                type:  map[string]int
string                          mode:  type
                                type:  string
int                             mode:  type
                                type:  int
m["hello, "+"world"]            mode:  value,assignable,ok
                                type:  (int, bool)
m                               mode:  value,addressable,assignable
                                type:  map[string]int
"hello, " + "world"             mode:  value
                                type:  string
                                value: "hello, world"
"hello, "                       mode:  value
                                type:  untyped string
                                value: "hello, "
"world"                         mode:  value
                                type:  untyped string
                                value: "world"
print(rune(v), ok)              mode:  void
                                type:  ()
print                           mode:  builtin
                                type:  func(rune, bool)
rune(v)                         mode:  value
                                type:  rune
rune                            mode:  type
                                type:  rune
...more not shown...
//!-output
*/
