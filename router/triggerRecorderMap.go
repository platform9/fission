package router

import (
	"time"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
	"log"
)

type (
	triggerRecorderMap struct {
		cache *cache.Cache // map[string]*crd.Recorder
	}
)

func makeTriggerRecorderMap(expiry time.Duration) *triggerRecorderMap {
	return &triggerRecorderMap{
		cache: cache.MakeCache(expiry, 0),
	}
}

func (trmap *triggerRecorderMap) lookup(trigger string) (*crd.Recorder, error) {
	item, err := trmap.cache.Get(trigger)
	if err != nil {
		return nil, err
	}
	u := item.(*crd.Recorder)
	return u, nil
}

func (trmap *triggerRecorderMap) assign(trigger string, recorder *crd.Recorder) {
	err, old := trmap.cache.Set(trigger, recorder)
	if err != nil {
		oldR := *(old.(*crd.Recorder))
		if (*recorder).Metadata.Name == oldR.Metadata.Name {
			return
		}
		log.Printf("error caching recorder for function name with a different value: %v", err)
	}

}

func (trmap *triggerRecorderMap) remove(trigger string) error {
	return trmap.cache.Delete(trigger)
}

