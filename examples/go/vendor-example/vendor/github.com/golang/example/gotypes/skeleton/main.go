// The skeleton command prints the boilerplate for a concrete type
// that implements the specified interface type.
//
// Example:
//	$ ./skeleton io ReadWriteCloser buffer
//	// *buffer implements io.ReadWriteCloser.
//	type buffer struct{ /* ... */ }
//	func (b *buffer) Close() error { panic("unimplemented") }
//	func (b *buffer) Read(p []byte) (n int, err error) { panic("unimplemented") }
//	func (b *buffer) Write(p []byte) (n int, err error) { panic("unimplemented") }
package main

import (
	"fmt"
	"go/types"
	"log"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/loader"
)

const usage = "Usage: skeleton <package> <interface> <concrete>"

//!+
func PrintSkeleton(pkg *types.Package, ifacename, concname string) error {
	obj := pkg.Scope().Lookup(ifacename)
	if obj == nil {
		return fmt.Errorf("%s.%s not found", pkg.Path(), ifacename)
	}
	if _, ok := obj.(*types.TypeName); !ok {
		return fmt.Errorf("%v is not a named type", obj)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return fmt.Errorf("type %v is a %T, not an interface",
			obj, obj.Type().Underlying())
	}
	// Use first letter of type name as receiver parameter.
	if !isValidIdentifier(concname) {
		return fmt.Errorf("invalid concrete type name: %q", concname)
	}
	r, _ := utf8.DecodeRuneInString(concname)

	fmt.Printf("// *%s implements %s.%s.\n", concname, pkg.Path(), ifacename)
	fmt.Printf("type %s struct{}\n", concname)
	mset := types.NewMethodSet(iface)
	for i := 0; i < mset.Len(); i++ {
		meth := mset.At(i).Obj()
		sig := types.TypeString(meth.Type(), (*types.Package).Name)
		fmt.Printf("func (%c *%s) %s%s {\n\tpanic(\"unimplemented\")\n}\n",
			r, concname, meth.Name(),
			strings.TrimPrefix(sig, "func"))
	}
	return nil
}

//!-

func isValidIdentifier(id string) bool {
	for i, r := range id {
		if !unicode.IsLetter(r) &&
			!(i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return id != ""
}

func main() {
	if len(os.Args) != 4 {
		log.Fatal(usage)
	}
	pkgpath, ifacename, concname := os.Args[1], os.Args[2], os.Args[3]

	// The loader loads a complete Go program from source code.
	var conf loader.Config
	conf.Import(pkgpath)
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err) // load error
	}
	pkg := lprog.Package(pkgpath).Pkg
	if err := PrintSkeleton(pkg, ifacename, concname); err != nil {
		log.Fatal(err)
	}
}

/*
//!+output1
$ ./skeleton io ReadWriteCloser buffer
// *buffer implements io.ReadWriteCloser.
type buffer struct{}
func (b *buffer) Close() error {
	panic("unimplemented")
}
func (b *buffer) Read(p []byte) (n int, err error) {
	panic("unimplemented")
}
func (b *buffer) Write(p []byte) (n int, err error) {
	panic("unimplemented")
}
//!-output1

//!+output2
$ ./skeleton net/http Handler myHandler
// *myHandler implements net/http.Handler.
type myHandler struct{}
func (m *myHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("unimplemented")
}
//!-output2
*/
