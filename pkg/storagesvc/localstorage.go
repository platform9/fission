package storagesvc

import (
	"os"

	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/local"
	uuid "github.com/satori/go.uuid"
)

type localStorage struct {
	storageType   string
	containerName string
	localPath     string
}

// NewLocalStorage return new local storage struct
func NewLocalStorage(localPath string) Storage {
	subdir := os.Getenv("SUBDIR")
	if len(subdir) == 0 {
		subdir = "fission-functions"
	}
	return localStorage{
		storageType:   StorageTypeLocal,
		containerName: subdir,
		localPath:     localPath,
	}
}

// Local
func (ls localStorage) getStorageType() string {
	return ls.storageType
}

func (ls localStorage) SetLocalPath(path string) {
	ls.localPath = path
}

func (ls localStorage) getUploadFileName() string {
	// This is not the item ID (that's returned by Put)
	// should we just use handler.Filename? what are the constraints here?
	return uuid.NewV4().String()
}

func (ls localStorage) getContainerName() string {
	return ls.containerName
}

func (ls localStorage) dial() (stow.Location, error) {
	cfg := stow.ConfigMap{"path": ls.localPath}
	return stow.Dial("local", cfg)
}
