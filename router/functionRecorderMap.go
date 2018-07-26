package router

import (
	"log"
	"time"

	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type (
	functionRecorderMap struct {
		cache *cache.Cache // map[string]*crd.Recorder
	}
)

// Why do we need an expiry?
func makeFunctionRecorderMap(expiry time.Duration) *functionRecorderMap {
	return &functionRecorderMap{
		cache: cache.MakeCache(expiry, 0),
	}
}

func (frmap *functionRecorderMap) lookup(function string) (*crd.Recorder, error) {
	item, err := frmap.cache.Get(function)
	if err != nil {
		return nil, err
	}
	u := item.(*crd.Recorder)
	return u, nil
}

func (frmap *functionRecorderMap) assign(function string, recorder *crd.Recorder) {
	err, old := frmap.cache.Set(function, recorder)
	if err != nil {
		oldR := *(old.(*crd.Recorder))
		if (*recorder).Metadata.Name == oldR.Metadata.Name {
			return
		}
		log.Printf("error caching recorder for function name with a different value: %v", err)
	}

}

func (frmap *functionRecorderMap) remove(function string) error {
	return frmap.cache.Delete(function)
}

