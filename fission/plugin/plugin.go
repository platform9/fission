// Package plugins provides support for creating extensible CLIs
//
// The plugin package contains four important structs
// - `plugin.Metadata` contains all the metadata of plugin (name, aliases, path, version...)
// - `plugin.Cache` is a in-memory memorization of plugins (with optional file-based persistence)
//
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	cmdTimeout      = 5 * time.Second
	cmdMetadataArgs = "--plugin"
)

var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginMetadataInvalid = errors.New("invalid plugin metadata")
	ErrPluginInvalid         = errors.New("invalid plugin")

	Prefix = fmt.Sprintf("%v-", os.Args[0])
	Cache  = &MetadataCache{}
)

// Metadata contains the metadata of a plugin.
// The only metadata that is guaranteed to be non-empty is the path and Name. All other fields are considered optional.
type Metadata struct {
	Name       string    `json:"name,omitempty"`
	Version    string    `json:"version,omitempty"`
	Url        string    `json:"url,omitempty"`
	Aliases    []string  `json:"aliases,omitempty"`
	Usage      string    `json:"usage,omitempty"`
	Path       string    `json:"path,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt,omitempty"`
}

func (md *Metadata) Alias(alias string) {
	if alias != md.Name && !md.HasAlias(alias) {
		md.Aliases = append(md.Aliases, alias)
	}
}

func (md *Metadata) HasAlias(needle string) bool {
	for _, alias := range md.Aliases {
		if alias == needle {
			return true
		}
	}
	return false
}

// Find searches the machine for the given plugin, returning the metadata of the plugin.
// The only metadata that is guaranteed to be non-empty is the path and Name. All other fields are considered optional.
// If found it returns the plugin, otherwise it returns ErrPluginNotFound if the plugin was not found or it returns
// ErrPluginMetadataInvalid if the plugin was found but considered unusable (e.g. not executable
// or invalid permissions).
func Find(pluginName string) (*Metadata, error) {
	// Look in MetadataCache for possible options (and aliases)
	name := pluginName
	if cached, ok := Cache.Get(pluginName); ok {
		name = cached.Name
	}

	// Search PATH for plugin as command-name
	// To check if plugin is actually there still.
	pluginPath, err := findPluginOnPath(name)
	if err != nil {
		Cache.Delete(pluginName)
		return nil, err
	}

	md, err := fetchPluginMetadata(pluginPath)
	if err != nil {
		Cache.Delete(pluginName)
		return nil, err
	}
	return md, nil
}

// Exec executes the plugin using the provided args.
// All input and output is redirected to stdin, stdout, and stderr.
func Exec(md *Metadata, args []string) error {
	cmd := exec.Command(md.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FindAll searches the machine for all plugins currently present.
func FindAll() map[string]*Metadata {
	plugins := map[string]*Metadata{}

	dirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range dirs {
		fs, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range fs {
			if !strings.HasPrefix(f.Name(), Prefix) {
				continue
			}
			fp := path.Join(dir, f.Name())
			md, err := fetchPluginMetadata(fp)
			if err != nil {
				continue
			}
			plugins[md.Name] = md
		}
	}
	Cache.Set(plugins)
	return plugins
}

func findPluginOnPath(pluginName string) (path string, err error) {
	binaryName := Prefix + pluginName
	path, _ = exec.LookPath(binaryName)

	if len(path) == 0 {
		return "", ErrPluginNotFound
	}
	return path, nil
}

// fetchPluginMetadata attempts to fetch the plugin metadata given the plugin path.
func fetchPluginMetadata(pluginPath string) (*Metadata, error) {
	// Before we check the MetadataCache, check if the plugin is actually present at the pluginPath
	d, err := os.Stat(pluginPath)
	if err != nil {
		return nil, ErrPluginNotFound
	}
	if m := d.Mode(); m.IsDir() || m&0111 == 0 {
		return nil, ErrPluginInvalid
	}

	// Check if we can retrieve the metadata for the plugin from the MetadataCache
	pluginName := strings.TrimPrefix(path.Base(pluginPath), Prefix)
	if c, ok := Cache.Get(pluginName); ok {
		if c.ModifiedAt == d.ModTime() {
			return c, nil
		}
	}

	// Fetch the metadata from the plugin itself.
	buf := bytes.NewBuffer(nil)
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pluginPath, cmdMetadataArgs) // Note: issue can occur with signal propagation
	cmd.Stdout = buf
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	// Parse metadata if possible
	md := &Metadata{}
	err = json.Unmarshal(buf.Bytes(), md)
	if err != nil {
		md.Name = pluginName
	}
	md.ModifiedAt = d.ModTime()
	md.Path = pluginPath
	md.Alias(pluginName)
	Cache.Write(md)
	return md, nil
}

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
