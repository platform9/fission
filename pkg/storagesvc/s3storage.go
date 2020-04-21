package storagesvc

import (
	"github.com/graymeta/stow"
	"github.com/graymeta/stow/s3"
)

type (
	S3Storage struct {
		name            string
		endpoint        string
		accessKeyId     string
		secretAccessKey string
		region          string
	}
)

func NewS3Storage(args ...string) Storage {
	return S3Storage{
		name:            StorageTypeS3,
		endpoint:        args[0],
		accessKeyId:     args[1],
		secretAccessKey: args[2],
		region:          args[3],
	}
}

func (ss S3Storage) getStorageType() string {
	return ss.name
}

func (st S3Storage) dial(localPath string) (stow.Location, error) {
	kind := "s3"
	config := stow.ConfigMap{
		s3.ConfigAccessKeyID: st.accessKeyId,
		s3.ConfigSecretKey:   st.secretAccessKey,
		s3.ConfigRegion:      st.region,
	}
	return stow.Dial(kind, config)
}
