package storagesvc

import (
	"os"
	"reflect"
	"testing"
)

func TestNewS3Storage(t *testing.T) {
	input := map[string]string{
		"bucketName":      "tmpBucket",
		"subDir":          "a/b/c",
		"accessKeyID":     "tmpAccessKeyID",
		"secretAccessKey": "tmpSecretAccessKey",
		"region":          "ap-south-1",
	}

	os.Setenv("STORAGE_S3_BUCKET_NAME", input["bucketName"])
	os.Setenv("STORAGE_S3_SUB_DIR", input["subDir"])
	os.Setenv("STORAGE_S3_ACCESS_KEY_ID", input["accessKeyID"])
	os.Setenv("STORAGE_S3_SECRET_ACCESS_KEY", input["secretAccessKey"])
	os.Setenv("STORAGE_S3_REGION", input["region"])

	storage := NewS3Storage().(s3Storage)

	for k, v := range input {
		valueInStruct := reflect.Indirect(reflect.ValueOf(storage)).FieldByName(k).String()

		// Test special case of adding suffix to bucket name
		if k == "bucketName" {
			v += "-fission-functions"
		}

		if valueInStruct != v {
			t.Errorf("Incorrect s3Storage field. Got: %s, Want %s", valueInStruct, v)
		}
	}

	if storage.storageType != StorageTypeS3 {
		t.Errorf("Incorrect storageType field. Got: %s, Want %s", storage.storageType, StorageTypeS3)
	}

	// TestGetStorageType
	if storage.getStorageType() != storage.storageType {
		t.Errorf("Incorrect getStorateType() method implementation. Got: %s, Want %s", storage.getStorageType(), storage.storageType)
	}

	// TestGetSubDir
	// Currently stow client doesn't support creating subDir within bucket. So we are using bucketName as subDir
	if storage.getSubDir() != storage.bucketName {
		t.Errorf("Incorrect getSubDir() method implementation. Got: %s, Want %s", storage.getSubDir(), storage.bucketName)
	}

}

func TestNewLocalStorage(t *testing.T) {
	storage := NewLocalStorage().(localStorage)

	// // When SUBDIR env is not set, expect a default "fission-functions" value.
	// if storage.subDir != "fission-functions" {
	// 	t.Errorf("Incorrect subDir field. Got: %s, Want %s", storage.subDir, "fission-functions")
	// }
	if storage.storageType != StorageTypeLocal {
		t.Errorf("Incorrect storageType field. Got: %s, Want %s", storage.storageType, StorageTypeLocal)
	}
}
