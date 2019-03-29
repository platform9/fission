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
	"io"
	"mime/multipart"
	"os"
	"time"

	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/local"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type (
	StorageType string

	storageConfig struct {
		storageType   StorageType
		localPath     string
		containerName string
		// other stuff, such as google or s3 credentials, bucket names etc
	}

	StowClient struct {
		logger    *zap.Logger
		config    *storageConfig
		location  stow.Location
		container stow.Container
	}
)

const (
	StorageTypeLocal StorageType = "local"
	PaginationSize   int         = 10
)

var (
	ErrNotFound                = errors.New("not found")
	ErrRetrievingItem          = errors.New("unable to retrieve item")
	ErrOpeningItem             = errors.New("unable to open item")
	ErrWritingFile             = errors.New("unable to write file")
	ErrWritingFileIntoResponse = errors.New("unable to copy item into http response")
)

func MakeStowClient(logger *zap.Logger, storageType StorageType, storagePath string, containerName string) (*StowClient, error) {
	if storageType != StorageTypeLocal {
		return nil, errors.New("Storage types other than 'local' are not implemented")
	}

	config := &storageConfig{
		storageType:   storageType,
		localPath:     storagePath,
		containerName: containerName,
	}

	stowClient := &StowClient{
		logger: logger.Named("stow_client"),
		config: config,
	}

	cfg := stow.ConfigMap{"path": config.localPath}
	loc, err := stow.Dial("local", cfg)
	if err != nil {
		return nil, err
	}
	stowClient.location = loc

	con, err := loc.CreateContainer(config.containerName)
	if os.IsExist(err) {
		var cons []stow.Container
		var cursor string

		// use location.Containers to find containers that match the prefix (container name)
		cons, cursor, err = loc.Containers(config.containerName, stow.CursorStart, 1)
		if err == nil {
			if !stow.IsCursorEnd(cursor) {
				// Should only have one storage container
				err = errors.New("Found more than one matched storage containers")
			} else {
				con = cons[0]
			}
		}
	}
	if err != nil {
		return nil, err
	}
	stowClient.container = con

	return stowClient, nil
}

// putFile writes the file on the storage
func (client *StowClient) putFile(file multipart.File, fileSize int64) (string, error) {
	// This is not the item ID (that's returned by Put)
	// should we just use handler.Filename? what are the constraints here?
	uploadName := uuid.NewV4().String()

	// save the file to the storage backend
	item, err := client.container.Put(uploadName, file, int64(fileSize), nil)
	if err != nil {
		client.logger.Error("error writing file on storage",
			zap.Error(err),
			zap.String("file", uploadName))
		return "", ErrWritingFile
	}

	client.logger.Debug("successfully wrote file on storage", zap.String("file", uploadName))
	return item.ID(), nil
}

// copyFileToStream gets the file contents into a stream
func (client *StowClient) copyFileToStream(fileId string, w io.Writer) error {
	item, err := client.container.Item(fileId)
	if err != nil {
		if err == stow.ErrNotFound {
			return ErrNotFound
		} else {
			return ErrRetrievingItem
		}
	}

	f, err := item.Open()
	if err != nil {
		return ErrOpeningItem
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		return ErrWritingFileIntoResponse
	}

	client.logger.Debug("successfully wrote file into httpresponse", zap.String("file", fileId))
	return nil
}

// removeFileByID deletes the file from storage
func (client *StowClient) removeFileByID(itemID string) error {
	return client.container.RemoveItem(itemID)
}

// filter defines an interface to filter out items from a set of items
type filter func(stow.Item, interface{}) bool

// This method returns all items in a container, filtering out items based on the filter function passed to it
func (client *StowClient) getItemIDsWithFilter(filterFunc filter, filterFuncParam interface{}) ([]string, error) {
	cursor := stow.CursorStart
	var items []stow.Item
	var err error

	archiveIDList := make([]string, 0)

	for {
		items, cursor, err = client.container.Items(stow.NoPrefix, cursor, PaginationSize)
		if err != nil {
			errors.Wrap(err, "error getting items from container")
			return nil, err
		}

		for _, item := range items {
			isItemFilterable := filterFunc(item, filterFuncParam)
			if isItemFilterable {
				continue
			}
			archiveIDList = append(archiveIDList, item.ID())
		}

		if stow.IsCursorEnd(cursor) {
			break
		}
	}

	return archiveIDList, nil
}

// filterItemCreatedAMinuteAgo is one type of filter function that filters out items created less than a minute ago.
// More filter functions can be written if needed, as long as they are of type filter
func (client StowClient) filterItemCreatedAMinuteAgo(item stow.Item, currentTime interface{}) bool {
	itemLastModTime, _ := item.LastMod()
	if currentTime.(time.Time).Sub(itemLastModTime) < 1*time.Minute {

		client.logger.Debug("item created less than a minute ago",
			zap.String("item", item.ID()),
			zap.Time("last_modified_time", itemLastModTime))
		return true
	}
	return false
}
