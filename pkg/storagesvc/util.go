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
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

func newStorageType(args ...string) (StorageType, error) {
	storageType := args[0]
	switch storageType {
	case "local":
		return strings.Join(args[1:], ","), nil
	case "s3":
		return strings.Join(args[1:], "-"), nil
	}
	return nil
}

func getStowLocation(config StorageConfig) (stow.Location, error) {
	return config.st.dial(config.st.localPath)
}

func getStorageTypeName(st StorageType) string {
	return st.name()
}

type StorageType interface {
	name() string
	dial()
}

type StorageTypeLocal struct {
	storageType string
}

func (st StorageTypeLocal) name() string {
	return constStorageTypeLocal
}

func (st StorageTypeLocal) dial(localPath StoragePath) (stow.Localtion, error) {
	cfg := stow.ConfigMap{"path": localPath}
	return stow.Dial("local", cfg)
}

type StorageTypeS3 struct {
	storageType     string
	endpoint        string
	accessKeyId     string
	secretAccessKey string
	region          string
}

func (st StorageTypeS3) name() string {
	return constStorageTypeS3
}

func (st StorageTypeS3) dial() {
	kind := "s3"
	config := stow.ConfigMap{
		s3.ConfigAccessKeyID: st.accessKeyId,
		s3.ConfigSecretKey:   st.secretAccessKey,
		s3.ConfigRegion:      st.region,
	}
	return stow.Dial(kind, config)
}

func MakeLocalStorage() StorageType {
	return StorageTypeLocal{
		storageType: STORAGE_TYPE_LOCAL,
	}
}

func MakeS3Storage() StorageType {
	return StorageTypeS3{
		storageType: STORAGE_TYPE_S3,
	}
}

func getQueryParamValue(urlString string, queryParam string) (string, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return "", errors.Wrapf(err, "error parsing URL string %q into URL", urlString)
	}
	return url.Query().Get(queryParam), nil
}

func getDifferenceOfLists(firstList []string, secondList []string) []string {
	tempMap := make(map[string]int)
	differenceList := make([]string, 0)

	for _, item := range firstList {
		tempMap[item] = 1
	}

	for _, item := range secondList {
		_, ok := tempMap[item]
		if ok {
			delete(tempMap, item)
		}
	}

	for k := range tempMap {
		differenceList = append(differenceList, k)
	}

	return differenceList
}
