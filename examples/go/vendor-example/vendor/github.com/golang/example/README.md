# Go example projects

This repository contains a collection of Go programs and libraries that
demonstrate the language, standard libraries, and tools.

## The examples

### [hello](hello/) ([godoc](//godoc.org/github.com/golang/example/hello)) and [stringutil](stringutil/) ([godoc](//godoc.org/github.com/golang/example/stringutil))

    go get github.com/golang/example/hello

A trivial "Hello, world" program that uses a stringutil package.

Command [hello](hello/) covers:

* The basic form of an executable command
* Importing packages (from the standard library and the local repository)
* Printing strings ([fmt](//golang.org/pkg/fmt/))

Library [stringutil](stringutil/) covers:

* The basic form of a library
* Conversion between string and []rune
* Table-driven unit tests ([testing](//golang.org/pkg/testing/))

### [outyet](outyet/) ([godoc](//godoc.org/github.com/golang/example/outyet))

    go get github.com/golang/example/outyet

A web server that answers the question: "Is Go 1.x out yet?"

Topics covered:

* Command-line flags ([flag](//golang.org/pkg/flag/))
* Web servers ([net/http](//golang.org/pkg/net/http/))
* HTML Templates ([html/template](//golang.org/pkg/html/template/))
* Logging ([log](//golang.org/pkg/log/))
* Long-running background processes
* Synchronizing data access between goroutines ([sync](//golang.org/pkg/sync/))
* Exporting server state for monitoring ([expvar](//golang.org/pkg/expvar/))
* Unit and integration tests ([testing](//golang.org/pkg/testing/))
* Dependency injection
* Time ([time](//golang.org/pkg/time/))

### [appengine-hello](appengine-hello/) ([godoc](//godoc.org/github.com/golang/example/appengine-hello))

	goapp get github.com/golang/example/appengine-hello

A trivial "Hello, world" App Engine application intended to be used as the
starting point for your own code.

_Note_: The `goapp` tool is part of the [Google App Engine SDK for Go](https://cloud.google.com/appengine/downloads#Google_App_Engine_SDK_for_Go).

### [gotypes](gotypes/) ([godoc](//godoc.org/github.com/golang/example/gotypes))

The `go/types` package is a type-checker for Go programs. It is one of the most
complex packages in Go's standard library, so we have provided this tutorial to
help you find your bearings. It comes with several example programs that you
can obtain using `go get` and play with as you learn to build tools that analyze
or manipulate Go programs.

### [template](template/) ([godoc](//godoc.org/github.com/golang/example/template))

A trivial web server that demonstrates the use of the
[`template` package](https://golang.org/pkg/text/template/)'s `block` feature.
