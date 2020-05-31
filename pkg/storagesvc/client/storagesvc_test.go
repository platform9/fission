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
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dchest/uniuri"
	"github.com/fission/fission/pkg/storagesvc"
	"github.com/minio/minio-go/v6"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"
	"go.uber.org/zap"
)

const (
	minioAccessKeyID     = "minioadmin"
	minioSecretAccessKey = "minioadmin"
	minioRegion          = "ap-south-1"
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

func runMinioDockerContainer(pool *dockertest.Pool) *dockertest.Resource {
	options := &dockertest.RunOptions{
		Repository: "minio/minio",
		Tag:        "latest",
		Cmd:        []string{"server", "/data"},
		PortBindings: map[dc.Port][]dc.PortBinding{
			"9000/tcp": {{HostIP: "", HostPort: "9000"}},
		},
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(options)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	return resource
}

func startS3StorageService(endpoint, bucketName, subDir string) {
	// testID := uniuri.NewLen(8)
	port := 8081
	enableArchivePruner := false

	logger, err := zap.NewDevelopment()
	panicIf(err)

	log.Println("starting storage svc")
	os.Setenv("STORAGE_S3_ENDPOINT", endpoint)
	os.Setenv("STORAGE_S3_BUCKET_NAME", bucketName)
	os.Setenv("STORAGE_S3_SUB_DIR", subDir)
	os.Setenv("STORAGE_S3_ACCESS_KEY_ID", minioAccessKeyID)
	os.Setenv("STORAGE_S3_SECRET_ACCESS_KEY", minioSecretAccessKey)
	os.Setenv("STORAGE_S3_REGION", minioRegion)

	storage := storagesvc.NewS3Storage()
	_ = storagesvc.RunStorageService(logger, storage, port, enableArchivePruner)

}

func TestS3StorageService(t *testing.T) {
	fmt.Println("Test S3 Storage service")
	var minioClient *minio.Client

	// Start minio docker container
	pool, err := dockertest.NewPool("")
	resource := runMinioDockerContainer(pool)

	endpoint := fmt.Sprintf("localhost:%s", resource.GetPort("9000/tcp"))

	if err := pool.Retry(func() error {
		minioClient, err = minio.New(endpoint, minioAccessKeyID, minioSecretAccessKey, false)
		if err != nil {
			return err
		}

		// This is to ensure container is up. Just getting minioClient
		// isn't suffcient to assume container is up.
		_, err = minioClient.ListBuckets()
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	defer pool.Purge(resource)

	// Start storagesvc
	bucketName := "test-s3-service"
	subDir := "x/y/z"
	startS3StorageService(endpoint, bucketName, subDir)

	time.Sleep(time.Second)
	client := MakeClient(fmt.Sprintf("http://localhost:%v/", 8081))

	// generate a test file
	tmpfile := MakeTestFile(10 * 1024)
	defer os.Remove(tmpfile.Name())

	// store it
	metadata := make(map[string]string)
	ctx := context.Background()
	fileID, err := client.Upload(ctx, tmpfile.Name(), &metadata)
	panicIf(err)

	time.Sleep(10 * time.Second)

	// Retrive file trhough minioClient
	reader, err := minioClient.GetObject(bucketName+"-fission-functions", fileID, minio.GetObjectOptions{})
	panicIf(err)
	defer reader.Close()

	retThroughMinio, err := ioutil.TempFile("", "storagesvc_verify_minio_")
	panicIf(err)
	defer os.Remove(retThroughMinio.Name())

	stat, err := reader.Stat()
	panicIf(err)

	if _, err := io.CopyN(retThroughMinio, reader, stat.Size); err != nil {
		log.Fatalln(err)
	}

	// Retrieve file through API
	retThroughAPI, err := ioutil.TempFile("", "storagesvc_verify_")
	panicIf(err)
	os.Remove(retThroughAPI.Name())

	err = client.Download(ctx, fileID, retThroughAPI.Name())
	panicIf(err)
	defer os.Remove(retThroughAPI.Name())

	// compare contents
	contentsMinio, err := ioutil.ReadFile(retThroughMinio.Name())
	panicIf(err)
	contentsAPI, err := ioutil.ReadFile(retThroughAPI.Name())
	panicIf(err)
	if !bytes.Equal(contentsMinio, contentsAPI) {
		log.Panic("Contents don't match")
	}

	// delete uploaded file
	err = client.Delete(ctx, fileID)
	panicIf(err)

	// make sure download fails
	err = client.Download(ctx, fileID, "xxx")
	if err == nil {
		log.Panic("Download succeeded but file isn't supposed to exist")
	}

}

func TestLocalStorageService(t *testing.T) {
	testID := uniuri.NewLen(8)
	port := 8080
	enableArchivePruner := false

	logger, err := zap.NewDevelopment()
	panicIf(err)

	log.Println("starting storage svc")
	localPath := fmt.Sprintf("/tmp/%v", testID)
	_ = os.Mkdir(localPath, os.ModePerm)
	storage := storagesvc.NewLocalStorage(fmt.Sprintf(localPath))
	_ = storagesvc.RunStorageService(logger, storage, port, enableArchivePruner)

	time.Sleep(time.Second)
	client := MakeClient(fmt.Sprintf("http://localhost:%v/", port))

	// generate a test file
	tmpfile := MakeTestFile(10 * 1024)
	defer os.Remove(tmpfile.Name())

	// store it
	metadata := make(map[string]string)
	ctx := context.Background()
	fileID, err := client.Upload(ctx, tmpfile.Name(), &metadata)
	panicIf(err)

	// make a temp file for verification
	retrievedfile, err := ioutil.TempFile("", "storagesvc_verify_")
	panicIf(err)
	os.Remove(retrievedfile.Name())

	// retrieve uploaded file
	err = client.Download(ctx, fileID, retrievedfile.Name())
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
	err = client.Delete(ctx, fileID)
	panicIf(err)

	// make sure download fails
	err = client.Download(ctx, fileID, "xxx")
	if err == nil {
		log.Panic("Download succeeded but file isn't supposed to exist")
	}

	// // cleanup /tmp
	os.RemoveAll(fmt.Sprintf("/tmp/%v", testID))
}
