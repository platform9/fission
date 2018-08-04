/*
Copyright 2016 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Command template is a trivial web server that uses the text/template (and
// html/template) package's "block" feature to implement a kind of template
// inheritance.
//
// It should be executed from the directory in which the source resides,
// as it will look for its template files in the current directory.
package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/image/", imageHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

// indexTemplate is the main site template.
// The default template includes two template blocks ("sidebar" and "content")
// that may be replaced in templates derived from this one.
var indexTemplate = template.Must(template.ParseFiles("index.tmpl"))

// Index is a data structure used to populate an indexTemplate.
type Index struct {
	Title string
	Body  string
	Links []Link
}

type Link struct {
	URL, Title string
}

// indexHandler is an HTTP handler that serves the index page.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	data := &Index{
		Title: "Image gallery",
		Body:  "Welcome to the image gallery.",
	}
	for name, img := range images {
		data.Links = append(data.Links, Link{
			URL:   "/image/" + name,
			Title: img.Title,
		})
	}
	if err := indexTemplate.Execute(w, data); err != nil {
		log.Println(err)
	}
}

// imageTemplate is a clone of indexTemplate that provides
// alternate "sidebar" and "content" templates.
var imageTemplate = template.Must(template.Must(indexTemplate.Clone()).ParseFiles("image.tmpl"))

// Image is a data structure used to populate an imageTemplate.
type Image struct {
	Title string
	URL   string
}

// imageHandler is an HTTP handler that serves the image pages.
func imageHandler(w http.ResponseWriter, r *http.Request) {
	data, ok := images[strings.TrimPrefix(r.URL.Path, "/image/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := imageTemplate.Execute(w, data); err != nil {
		log.Println(err)
	}
}

// images specifies the site content: a collection of images.
var images = map[string]*Image{
	"go":     {"The Go Gopher", "https://golang.org/doc/gopher/frontpage.png"},
	"google": {"The Google Logo", "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png"},
}
