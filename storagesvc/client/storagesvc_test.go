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

package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dchest/uniuri"
	"go.uber.org/zap"

	"github.com/fission/fission/storagesvc"
)

func panicIf(err error) {
	if err != nil {
		log.Panicf("Error: %v", err)
	}
}

func MakeTestFile(size int) *os.File {
	f, err := ioutil.TempFile("", "storagesvc_test_")
	panicIf(err)

	_, err = f.Write(bytes.Repeat([]byte("."), size))
	panicIf(err)

	return f
}

func TestStorageService(t *testing.T) {
	testId := uniuri.NewLen(8)
	port := 8080
	enableArchivePruner := false

	logger, err := zap.NewDevelopment()
	panicIf(err)

	log.Println("starting storage svc")
	_ = storagesvc.RunStorageService(
		logger, storagesvc.StorageTypeLocal, "/tmp", testId, port, enableArchivePruner)

	time.Sleep(time.Second)
	client := MakeClient(fmt.Sprintf("http://localhost:%v/", port))

	// generate a test file
	tmpfile := MakeTestFile(10 * 1024)
	defer os.Remove(tmpfile.Name())

	// store it
	metadata := make(map[string]string)
	ctx := context.Background()
	fileId, err := client.Upload(ctx, tmpfile.Name(), &metadata)
	panicIf(err)

	// make a temp file for verification
	retrievedfile, err := ioutil.TempFile("", "storagesvc_verify_")
	panicIf(err)
	os.Remove(retrievedfile.Name())

	// retrieve uploaded file
	err = client.Download(ctx, fileId, retrievedfile.Name())
	panicIf(err)
	defer os.Remove(retrievedfile.Name())

	// compare contents
	contents1, err := ioutil.ReadFile(tmpfile.Name())
	panicIf(err)
	contents2, err := ioutil.ReadFile(retrievedfile.Name())
	panicIf(err)
	if !bytes.Equal(contents1, contents2) {
		log.Panic("Contents don't match")
	}

	// delete uploaded file
	err = client.Delete(ctx, fileId)
	panicIf(err)

	// make sure download fails
	err = client.Download(ctx, fileId, "xxx")
	if err == nil {
		log.Panic("Download succeeded but file isn't supposed to exist")
	}

	// cleanup /tmp
	os.RemoveAll(fmt.Sprintf("/tmp/%v", testId))
}
