package storagesvc

import (
	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/local"
)

type LocalStorage struct {
	name string
}

func NewLocalStorage() Storage {
	return LocalStorage{
		name: StorageTypeLocal,
	}
}

// Local
func (ls LocalStorage) getStorageType() string {
	return ls.name
}

func (ls LocalStorage) dial(localPath string) (stow.Location, error) {
	cfg := stow.ConfigMap{"path": localPath}
	return stow.Dial("local", cfg)
}
