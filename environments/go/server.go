package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"plugin"

	// DO NOT IMPORT THIRD PARTY PACKAGES
	// The 3rd party package version used by go server may be
	// different from the one in user's source code and will
	// cause plugin version mismatched. Hence, we should never
	// import any external packages except the Fission or built-in
	// packages.

	"github.com/fission/fission/environments/go/context"
)

const (
	CODE_PATH = "/userfunc/user"
)

type (
	FunctionLoadRequest struct {
		// FilePath is an absolute filesystem path to the
		// function. What exactly is stored here is
		// env-specific. Optional.
		FilePath string `json:"filepath"`

		// FunctionName has an environment-specific meaning;
		// usually, it defines a function within a module
		// containing multiple functions. Optional; default is
		// environment-specific.
		FunctionName string `json:"functionName"`

		// URL to expose this function at. Optional; defaults
		// to "/".
		URL string `json:"url"`
	}
)

var userFunc http.HandlerFunc

func loadPlugin(codePath, entrypoint string) (http.HandlerFunc, error) {

	// if codepath's a directory, load the file inside it
	info, err := os.Stat(codePath)
	if err != nil {
		return nil, fmt.Errorf("error checking plugin path: %v", err)
	}
	if info.IsDir() {
		files, err := ioutil.ReadDir(codePath)
		if err != nil {
			return nil, fmt.Errorf("error reading directory: %v", err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no files to load: %v", codePath)
		}
		fi := files[0]
		codePath = filepath.Join(codePath, fi.Name())
	}

	log.Printf("loading plugin from %v", codePath)
	p, err := plugin.Open(codePath)
	if err != nil {
		return nil, fmt.Errorf("error loading plugin: %v", err)
	}
	sym, err := p.Lookup(entrypoint)
	if err != nil {
		return nil, fmt.Errorf("entry point not found: %v", err)
	}

	switch h := sym.(type) {
	case *http.Handler:
		return (*h).ServeHTTP, nil
	case *http.HandlerFunc:
		return *h, nil
	case func(http.ResponseWriter, *http.Request):
		return h, nil
	case func(context.Context, http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request) {
			c := context.New()
			h(c, w, r)
		}, nil
	default:
		panic("Entry point not found: bad type")
	}
}

func specializeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFunc != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Not a generic container"))
			return
		}

		_, err := os.Stat(CODE_PATH)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("code path (%v) does not exist: %v", CODE_PATH, err)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(CODE_PATH + ": not found"))
				return
			} else {
				log.Printf("unknown error looking for code path(%v): %v", CODE_PATH, err)
				err = fmt.Errorf("unknown error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
		}

		log.Println("specializing ...")
		userFunc, err = loadPlugin(CODE_PATH, "Handler")
		if err != nil {
			err = fmt.Errorf("error specializing function: %v", err)
			log.Printf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		log.Println("done")
	}
}

func specializeHandlerV2() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFunc != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Not a generic container"))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var loadreq FunctionLoadRequest
		err = json.Unmarshal(body, &loadreq)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = os.Stat(loadreq.FilePath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("code path (%v) does not exist: %v", loadreq.FilePath, err)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(loadreq.FilePath + ": not found"))
				return
			} else {
				log.Printf("unknown error looking for code path(%v): %v", loadreq.FilePath, err)
				err = fmt.Errorf("unknown error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
		}

		log.Println("specializing ...")
		userFunc, err = loadPlugin(loadreq.FilePath, loadreq.FunctionName)
		if err != nil {
			err = fmt.Errorf("error specializing function: %v", err)
			log.Printf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		log.Println("done")
	}
}

func readinessProbeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {

	http.HandleFunc("/healthz", readinessProbeHandler)
	http.HandleFunc("/specialize", specializeHandler())
	http.HandleFunc("/v2/specialize", specializeHandlerV2())

	// Generic route -- all http requests go to the user function.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if userFunc == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Generic container: no requests supported"))
			return
		}
		userFunc(w, r)
	})

	log.Println("listening on 8888 ...")
	http.ListenAndServe(":8888", nil)
}
