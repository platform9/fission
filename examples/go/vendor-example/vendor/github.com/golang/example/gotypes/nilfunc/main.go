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

import "bytes"

func main() {
	var buf bytes.Buffer
	if buf.Bytes == nil && bytes.Repeat != nil && main == nil {
		// ...
	}
}
`

//!-input

var fset = token.NewFileSet()

func main() {
	f, err := parser.ParseFile(fset, "input.go", input, 0)
	if err != nil {
		log.Fatal(err) // parse error
	}
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	if _, err = conf.Check("cmd/hello", fset, []*ast.File{f}, info); err != nil {
		log.Fatal(err) // type error
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n != nil {
			CheckNilFuncComparison(info, n)
		}
		return true
	})
}

//!+
// CheckNilFuncComparison reports unintended comparisons
// of functions against nil, e.g., "if x.Method == nil {".
func CheckNilFuncComparison(info *types.Info, n ast.Node) {
	e, ok := n.(*ast.BinaryExpr)
	if !ok {
		return // not a binary operation
	}
	if e.Op != token.EQL && e.Op != token.NEQ {
		return // not a comparison
	}

	// If this is a comparison against nil, find the other operand.
	var other ast.Expr
	if info.Types[e.X].IsNil() {
		other = e.Y
	} else if info.Types[e.Y].IsNil() {
		other = e.X
	} else {
		return // not a comparison against nil
	}

	// Find the object.
	var obj types.Object
	switch v := other.(type) {
	case *ast.Ident:
		obj = info.Uses[v]
	case *ast.SelectorExpr:
		obj = info.Uses[v.Sel]
	default:
		return // not an identifier or selection
	}

	if _, ok := obj.(*types.Func); !ok {
		return // not a function or method
	}

	fmt.Printf("%s: comparison of function %v %v nil is always %v\n",
		fset.Position(e.Pos()), obj.Name(), e.Op, e.Op == token.NEQ)
}

//!-

/*
//!+output
$ go build github.com/golang/example/gotypes/nilfunc
$ ./nilfunc
input.go:7:5: comparison of function Bytes == nil is always false
input.go:7:25: comparison of function Repeat != nil is always true
input.go:7:48: comparison of function main == nil is always false
//!-output
*/
