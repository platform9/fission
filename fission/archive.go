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

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	storageSvcClient "github.com/fission/fission/storagesvc/client"
)

// downloadStoragesvcURL downloads and return archive content with given storage service url
func downloadStoragesvcURL(client *client.Client, fileUrl string) io.ReadCloser {
	u, err := url.ParseRequestURI(fileUrl)
	if err != nil {
		return nil
	}
	// replace in-cluster storage service host with controller server url
	fileDownloadUrl := strings.TrimSuffix(client.Url, "/") + "/proxy/storage" + u.RequestURI()
	reader, err := downloadURL(fileDownloadUrl)
	checkErr(err, fmt.Sprintf("download from storage service url: %v", fileUrl))
	return reader
}

// upload a file and return a fission.Archive
func createArchive(client *client.Client, fileName string) *fission.Archive {
	var archive fission.Archive

	// fetch archive from arbitrary url if fileName is a url
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		fileName = downloadToTempFile(fileName)
	}

	if fileSize(fileName) < fission.ArchiveLiteralSizeLimit {
		contents := getContents(fileName)
		archive.Type = fission.ArchiveTypeLiteral
		archive.Literal = contents
	} else {
		u := strings.TrimSuffix(client.Url, "/") + "/proxy/storage"
		ssClient := storageSvcClient.MakeClient(u)

		// TODO add a progress bar
		id, err := ssClient.Upload(fileName, nil)
		checkErr(err, fmt.Sprintf("upload file %v", fileName))

		archiveUrl := ssClient.GetUrl(id)

		archive.Type = fission.ArchiveTypeUrl
		archive.URL = archiveUrl

		f, err := os.Open(fileName)
		if err != nil {
			checkErr(err, fmt.Sprintf("find file %v", fileName))
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			checkErr(err, fmt.Sprintf("calculate checksum for file %v", fileName))
		}

		archive.Checksum = fission.Checksum{
			Type: fission.ChecksumTypeSHA256,
			Sum:  hex.EncodeToString(h.Sum(nil)),
		}
	}
	return &archive
}

func writeArchiveToFile(fileName string, reader io.Reader) error {
	tmpDir, err := getTempDir()
	if err != nil {
		return err
	}

	path := filepath.Join(tmpDir, fileName+".tmp")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}
	err = os.Chmod(path, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(path, fileName)
	if err != nil {
		return err
	}

	return nil
}

// downloadToTempFile fetches archive file from arbitrary url
// and write it to temp file for further usage
func downloadToTempFile(fileUrl string) string {
	reader, err := downloadURL(fileUrl)
	defer reader.Close()
	checkErr(err, fmt.Sprintf("download from url: %v", fileUrl))

	tmpDir, err := getTempDir()
	checkErr(err, "create temp directory")

	tmpFilename := uuid.NewV4().String()
	destination := filepath.Join(tmpDir, tmpFilename)
	err = os.Mkdir(tmpDir, 0744)
	checkErr(err, "create temp directory")

	err = writeArchiveToFile(destination, reader)
	checkErr(err, "write archive to file")

	return destination
}

// downloadURL downloads file from given url
func downloadURL(fileUrl string) (io.ReadCloser, error) {
	resp, err := http.Get(fileUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v - HTTP response returned non 200 status", resp.StatusCode)
	}
	return resp.Body, nil
}
