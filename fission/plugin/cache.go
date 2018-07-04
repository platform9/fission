package plugin

import (
	"io/ioutil"
	"os"
)

// MetadataCache allows short-circuiting of plugin lookups by memorizing plugin states.
// By default the Cache will Cache the results in the memory.
// If path is specified the Cache will also be cached persistently as a JSON file.
type MetadataCache struct {
	path  string
	inMem map[string]*Metadata
}

func NewCache(cachePath string) *MetadataCache {
	return &MetadataCache{
		path: cachePath,
	}
}

func (c MetadataCache) Delete(cachedPluginName string) error {
	mds := c.Entries()
	cached, ok := mds[cachedPluginName]
	if !ok {
		return nil
	}
	for _, alias := range cached.Aliases {
		if cachedAlias, ok := mds[alias]; ok && cachedAlias.Name == cached.Name {
			delete(mds, alias)
		}
	}
	delete(mds, cachedPluginName)
	return c.Set(mds)
}

func (c MetadataCache) Get(key string) (*Metadata, bool) {
	if c.inMem == nil {
		c.loadFileCache()
	}
	val, ok := c.inMem[key]
	return val, ok
}

func (c MetadataCache) Entries() map[string]*Metadata {
	if c.inMem == nil {
		c.loadFileCache()
	}
	return removeAliases(c.inMem)
}

func (c MetadataCache) Write(md *Metadata) error {
	cached := c.Entries()
	if existing, ok := cached[md.Name]; ok {
		for _, alias := range existing.Aliases {
			md.Alias(alias)
		}
	}
	cached[md.Name] = md
	return c.Set(cached)
}

func (c *MetadataCache) Clear() {
	// Clear in-memory MetadataCache
	c.inMem = nil

	// Remove MetadataCache file, if set.
	if len(c.path) != 0 {
		os.Remove(c.path)
	}
}

// loadFileCache loads the Cache from the persisted Cache on the filesystem.
// If present the Cache replaces the current in-memory Cache.
// If not present, an empty Cache or an error will be returned.
func (c *MetadataCache) loadFileCache() error {
	cached := map[string]*Metadata{}
	if len(c.path) == 0 {
		return nil
	}
	bs, err := ioutil.ReadFile(c.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bs, &cached)
	if err != nil {
		return err
	}
	c.inMem = expandAliases(cached)
	return nil
}

func (c MetadataCache) Set(mds map[string]*Metadata) error {
	// Store in file if set
	if len(c.path) != 0 {
		bs, err := json.Marshal(mds)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(c.path, bs, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Load into in-memory cache
	c.loadFileCache()
	return nil
}

func removeAliases(mp map[string]*Metadata) map[string]*Metadata {
	entries := map[string]*Metadata{}
	for _, v := range mp {
		entries[v.Name] = v
	}
	return entries
}

func expandAliases(mds map[string]*Metadata) map[string]*Metadata {
	entries := map[string]*Metadata{}
	for _, md := range mds {
		for _, alias := range md.Aliases {
			if _, ok := mds[alias]; ok {
				continue
			}
			entries[alias] = md
		}
	}
	return entries
}
