/*
Copyright 2017 The Fission Authors.

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

package storagesvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fission/fission"
	"github.com/gorilla/mux"
	_ "github.com/graymeta/stow/local"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
)

type (
	StorageService struct {
		storageClient *StowClient
		port          int
	}

	UploadResponse struct {
		ID string `json:"id"`
	}
)

// Handle multipart file uploads.
func (ss *StorageService) uploadHandler(w http.ResponseWriter, r *http.Request) {
	// handle upload
	r.ParseMultipartForm(0)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		http.Error(w, "missing upload file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// stow wants the file size, but that's different from the
	// content length, the content length being the size of the
	// encoded file in the HTTP request. So we require an
	// "X-File-Size" header in bytes.

	fileSizeS, ok := r.Header["X-File-Size"]
	if !ok {
		log.Error("Missing X-File-Size")
		http.Error(w, "missing X-File-Size header", http.StatusBadRequest)
		return
	}

	fileSize, err := strconv.Atoi(fileSizeS[0])
	if err != nil {
		log.WithError(err).Errorf("Error parsing x-file-size: '%v'", fileSizeS)
		http.Error(w, "missing or bad X-File-Size header", http.StatusBadRequest)
		return
	}

	// TODO: allow headers to add more metadata (e.g. environment
	// and function metadata)
	log.Infof("Handling upload for %v", handler.Filename)
	//fileMetadata := make(map[string]interface{})
	//fileMetadata["filename"] = handler.Filename

	id, err := ss.storageClient.putFile(file, int64(fileSize))
	if err != nil {
		log.WithError(err).Error("Error saving uploaded file")
		http.Error(w, "Error saving uploaded file", http.StatusInternalServerError)
		return
	}

	// respond with an ID that can be used to retrieve the file
	ur := &UploadResponse{
		ID: id,
	}
	resp, err := json.Marshal(ur)
	if err != nil {
		http.Error(w, "Error marshaling response", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func (ss *StorageService) getIdFromRequest(r *http.Request) (string, error) {
	values := r.URL.Query()
	ids, ok := values["id"]
	if !ok || len(ids) == 0 {
		return "", errors.New("Missing `id' query param")
	}
	return ids[0], nil
}

func (ss *StorageService) deleteHandler(w http.ResponseWriter, r *http.Request) {
	// get id from request
	fileId, err := ss.getIdFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = ss.storageClient.removeFileByID(fileId)
	if err != nil {
		msg := fmt.Sprintf("Error deleting item: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (ss *StorageService) downloadHandler(w http.ResponseWriter, r *http.Request) {
	// get id from request
	fileId, err := ss.getIdFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file (called "item" in stow's jargon), open it,
	// stream it to response
	err = ss.storageClient.copyFileToStream(fileId, w)
	if err != nil {
		log.WithError(err).Errorf("Error getting item id '%v'", fileId)
		if err == ErrNotFound {
			http.Error(w, "Error retrieving item: not found", http.StatusNotFound)
		} else if err == ErrRetrievingItem {
			http.Error(w, "Error retrieving item", http.StatusBadRequest)
		} else if err == ErrOpeningItem {
			http.Error(w, "Error opening item", http.StatusBadRequest)
		} else if err == ErrWritingFileIntoResponse {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
		return
	}
}

func (ss *StorageService) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func MakeStorageService(storageClient *StowClient, port int) *StorageService {
	return &StorageService{
		storageClient: storageClient,
		port:          port,
	}
}

func (ss *StorageService) Start(port int) {
	r := mux.NewRouter()
	r.HandleFunc("/v1/archive", ss.uploadHandler).Methods("POST")
	r.HandleFunc("/v1/archive", ss.downloadHandler).Methods("GET")
	r.HandleFunc("/v1/archive", ss.deleteHandler).Methods("DELETE")
	r.HandleFunc("/healthz", ss.healthHandler).Methods("GET")

	address := fmt.Sprintf(":%v", port)

	r.Use(fission.LoggingMiddleware)
	err := http.ListenAndServe(address, &ochttp.Handler{
		Handler: r,
		// Propagation: &b3.HTTPFormat{},
	})

	log.Fatal(err)
}

func RunStorageService(storageType StorageType, storagePath string, containerName string, port int, enablePruner bool) *StorageService {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	// initialize logger
	log.SetLevel(log.InfoLevel)

	// create a storage client
	storageClient, err := MakeStowClient(storageType, storagePath, containerName)
	if err != nil {
		log.Fatalf("Error creating stowClient: %v", err)
	}

	// create http handlers
	storageService := MakeStorageService(storageClient, port)
	go storageService.Start(port)

	// enablePruner prevents storagesvc unit test from needing to talk to kubernetes
	if enablePruner {
		// get the prune interval and start the archive pruner
		pruneInterval, err := strconv.Atoi(os.Getenv("PRUNE_INTERVAL"))
		if err != nil {
			pruneInterval = defaultPruneInterval
		}
		pruner, err := MakeArchivePruner(storageClient, time.Duration(pruneInterval))
		if err != nil {
			log.Fatalf("Error creating archivePruner: %v", err)
		}
		go pruner.Start()
	}

	log.Info("Storage service started")
	return storageService
}
