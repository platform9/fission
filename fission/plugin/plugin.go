// Package plugins provides support for creating extensible CLIs
//
// The plugin package contains four important structs
// - `plugin.Metadata` contains all the metadata of plugin (name, aliases, path, version...)
// - `plugin.Cache` is a in-memory memorization of plugins (with optional file-based persistence)
// - `plugin.Registry` is a simple representation of a remote registry. The manager uses these to search for plugins
// to suggest to the user to install.
// - `plugin.Manager` ties everything together, providing the API user to find, search, list, and execute plugins.
//
// Unless you want to modify specific behavior of the plugin functionality, you simply can use the public
// package-level functions that are mapped to the functions of a default plugin.Manager.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	cmdTimeout           = 5 * time.Second
	cmdMetadataArgs      = "--plugin"
	cacheRefreshInterval = 1 * time.Hour
)

var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginMetadataInvalid = errors.New("invalid plugin metadata")
	ErrPluginInvalid         = errors.New("invalid plugin")
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

var DefaultManager = &Manager{
	Prefix: fmt.Sprintf("%v-", os.Args[0]),
	Cache:  &Cache{},
}

func Find(pluginName string) (*Metadata, error) {
	return DefaultManager.Find(pluginName)
}

func Exec(md *Metadata, args []string) error {
	return DefaultManager.Exec(md, args)
}

func FindAll() map[string]*Metadata {
	return DefaultManager.FindAll()
}

func SetCache(path string) {
	DefaultManager.Cache.Clear()
	if DefaultManager.Cache == nil {
		DefaultManager.Cache = NewCache(path)
	}
	DefaultManager.Cache.path = path
}

func SetPrefix(prefix string) {
	DefaultManager.Prefix = prefix
}

func AddRegistry(registry Registry) {
	DefaultManager.Registries = append(DefaultManager.Registries, registry)
}

func EnsureFreshCache() {
	if DefaultManager.Cache.IsStale() {
		DefaultManager.FindAll()
	}
}

func ClearCache() {
	DefaultManager.Cache.Clear()
}

func SearchRegistries(pluginName string) (*Metadata, Registry, error) {
	return DefaultManager.SearchRegistries(pluginName)
}

func Validate(md *Metadata) error {
	if md == nil {
		logrus.Debug("plugin metadata %v invalid: %v", md, "metadata is nil")
		return ErrPluginMetadataInvalid
	}
	if len(md.Name) == 0 {
		logrus.Debug("plugin metadata %v invalid: %v", md, "empty name")
		return ErrPluginMetadataInvalid
	}
	if len(md.Path) == 0 {
		logrus.Debug("plugin metadata %v invalid: %v", md, "empty path")
		return ErrPluginMetadataInvalid
	}
	d, err := os.Stat(md.Path)
	if err != nil {
		return ErrPluginNotFound
	}
	if m := d.Mode(); m.IsDir() || m&0111 == 0 {
		return ErrPluginInvalid
	}
	return nil
}

type Manager struct {
	Prefix     string
	Registries []Registry
	Cache      *Cache // Empty means: do not Cache
}

// Find searches the machine for the given plugin, returning the metadata of the plugin.
// The only metadata that is guaranteed to be non-empty is the path and Name. All other fields are considered optional.
// If found it returns the plugin, otherwise it returns ErrPluginNotFound if the plugin was not found or it returns
// ErrPluginMetadataInvalid if the plugin was found but considered unusable (e.g. not executable
// or invalid permissions).
func (mgr *Manager) Find(pluginName string) (*Metadata, error) {
	// Look in Cache for possible options (and aliases)
	name := pluginName
	if cached, ok := mgr.Cache.Get(pluginName); ok {
		name = cached.Name
	}

	// Search PATH for plugin as command-name
	// To check if plugin is actually there still.
	pluginPath, err := mgr.findPluginOnPath(name)
	if err != nil {
		return nil, err
	}

	md, err := mgr.fetchPluginMetadata(pluginPath)
	if err != nil {
		return nil, err
	}
	return md, nil
}

// Exec executes the plugin using the provided args.
// All input and output is redirected to stdin, stdout, and stderr.
func (mgr *Manager) Exec(md *Metadata, args []string) error {
	if err := Validate(md); err != nil {
		return err
	}
	cmd := exec.Command(md.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SearchRegistries searches the registries for the plugin.
// If found it returns the plugin metadata and the registry where it was found.
// Otherwise it returns ErrPluginNotFound.
func (mgr *Manager) SearchRegistries(pluginName string) (*Metadata, Registry, error) {
	for _, registryPath := range mgr.Registries {
		registry, err := registryPath.FetchAll()
		if err != nil {
			logrus.Debug(err)
			continue
		}

		// Check if registry contains the plugin we are looking for.
		for k, p := range registry {
			// If the name is not set, the key is assumed to be the name
			if len(p.Name) == 0 {
				p.Name = k
			}
			if p.Name == pluginName {
				return p, registryPath, nil
			}
		}
	}
	return nil, "", ErrPluginNotFound
}

// FindAll searches the machine for all plugins currently present.
func (mgr *Manager) FindAll() map[string]*Metadata {
	plugins := map[string]*Metadata{}

	dirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range dirs {
		fs, err := ioutil.ReadDir(dir)
		if err != nil {
			logrus.Debugf("Failed to read $PATH directory %v: %v", dir, err)
			continue
		}
		for _, f := range fs {
			if !strings.HasPrefix(f.Name(), mgr.Prefix) {
				continue
			}
			fp := path.Join(dir, f.Name())
			md, err := mgr.fetchPluginMetadata(fp)
			if err != nil {
				logrus.Debugf("Failed to fetch metadata for %v: %v", f.Name(), err)
				continue
			}
			plugins[md.Name] = md
		}
	}
	err := mgr.Cache.WriteAll(plugins)
	if err != nil {
		logrus.Debug("Failed to Cache plugin metadata: %v", err)
	}
	return plugins
}

func (mgr *Manager) fetchRegistry(registryPath string) (map[string]*Metadata, error) {
	var registry map[string]*Metadata
	var bs []byte
	var err error
	if strings.HasPrefix(registryPath, "/") {
		bs, err = ioutil.ReadFile(registryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch filesystem registry %v: %v", registryPath, err)
		}
	} else {
		resp, err := http.Get(registryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch HTTP registry %v: %v", registry, err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to fetch HTTP registry %v: %v", registry, resp.Status)
		}
		bs, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read HTTP registry %v: %v", registry, resp.Status)
		}
		resp.Body.Close()
	}
	err = yaml.Unmarshal(bs, &registry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry %v: %v", registry, err)
	}
	return registry, nil
}

func (mgr *Manager) findPluginOnPath(pluginName string) (path string, err error) {
	binaryName := mgr.pluginNameToFilename(pluginName)
	path, err = exec.LookPath(binaryName)
	if err != nil {
		logrus.Debugf("Plugin not found on $PATH: %v", err)
	}

	if len(path) == 0 {
		return "", ErrPluginNotFound
	}
	return path, nil
}

// fetchPluginMetadata attempts to fetch the plugin metadata given the plugin path.
func (mgr *Manager) fetchPluginMetadata(pluginPath string) (*Metadata, error) {
	// Before we check the Cache, check if the plugin is actually present at the pluginPath
	d, err := os.Stat(pluginPath)
	if err != nil {
		return nil, ErrPluginNotFound
	}
	if m := d.Mode(); m.IsDir() || m&0111 == 0 {
		return nil, ErrPluginInvalid
	}

	// Check if we can retrieve the metadata for the plugin from the Cache
	if err != nil {
		logrus.Debugf("Failed to read Cache for metadata fetch of %v: %v", pluginPath, err)
	}
	if c, ok := mgr.Cache.Get(mgr.filenameToPluginName(path.Base(pluginPath))); ok {
		if c.ModifiedAt == d.ModTime() {
			return c, nil
		} else {
			logrus.Debugf("Cache entry for %v is stale; refreshing.", pluginPath)
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
		logrus.Debugf("Failed to read plugin metadata: %v", err)
		md.Name = mgr.filenameToPluginName(path.Base(pluginPath))
	}
	md.ModifiedAt = d.ModTime()
	md.Path = pluginPath
	mgr.Cache.Write(md)
	return md, nil
}

func (mgr *Manager) pluginNameToFilename(pluginName string) string {
	return mgr.Prefix + pluginName
}

func (mgr *Manager) filenameToPluginName(binaryName string) string {
	return strings.TrimPrefix(binaryName, mgr.Prefix)
}

// Cache allows short-circuiting of plugin lookups by memorizing plugin states.
// By default the cache will cache the results in the memory.
// If path is specified the cache will also be cached persistently as a JSON file.
type Cache struct {
	path        string
	inMem       map[string]*Metadata
	lastUpdated time.Time
}

func NewCache(cachePath string) *Cache {
	return &Cache{
		path: cachePath,
	}
}

func (c *Cache) IsStale() bool {
	if c == nil {
		// If there is no Cache it cannot be stale.
		return false
	}
	modTime := c.lastUpdated
	if modTime != (time.Time{}) {
		fi, err := os.Stat(c.path)
		if err != nil {
			logrus.Debugf("Failed to stat Cache for staleness: %v", err)
			return true
		}
		modTime = fi.ModTime()
	}
	return time.Now().After(modTime.Add(cacheRefreshInterval))
}

func (c *Cache) Delete(cachedPluginName string) error {
	if c == nil {
		return nil
	}
	mds, err := c.Entries()
	if err != nil {
		return err
	}
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
	return c.WriteAll(mds)
}

func (c *Cache) Get(key string) (*Metadata, bool) {
	if c == nil {
		return nil, false
	}
	if c.inMem == nil {
		c.loadFileCache()
	}
	val, ok := c.inMem[key]
	return val, ok
}

func (c *Cache) Entries() (map[string]*Metadata, error) {
	if c == nil {
		return map[string]*Metadata{}, nil
	}

	if c.inMem == nil {
		if _, err := c.loadFileCache(); err != nil {
			return nil, err
		}
	}

	return removeAliases(c.inMem), nil
}

func (c *Cache) Write(md *Metadata) error {
	if c == nil {
		return nil
	}
	cached, err := c.Entries()
	if err != nil {
		return err
	}
	cached[md.Name] = md
	err = c.WriteAll(cached)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Clear() {
	if c == nil {
		return
	}
	// Clear in-memory Cache
	c.inMem = nil

	// Remove Cache file, if set.
	if len(c.path) != 0 {
		err := os.Remove(c.path)
		if err != nil {
			logrus.Debugf("Failed to Delete Cache at %v: %v", c.path, err)
		}
	}
}

// loadFileCache loads the cache from the persisted cache on the filesystem.
// If present the cache replaces the current in-memory cache.
// If not present, an empty cache or an error will be returned.
func (c *Cache) loadFileCache() (map[string]*Metadata, error) {
	cached := map[string]*Metadata{}
	if len(c.path) == 0 {
		return cached, nil
	}
	bs, err := ioutil.ReadFile(c.path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bs, &cached)
	if err != nil {
		return nil, err
	}
	c.inMem = expandAliases(cached)

	logrus.Debugf("Read plugin metadata from cache at %v", c.path)
	return cached, nil
}

func (c *Cache) WriteAll(mds map[string]*Metadata) error {
	if c == nil {
		return nil
	}

	// Store in Cache file if set
	if len(c.path) != 0 {
		bs, err := json.Marshal(mds)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(c.path, bs, os.ModePerm)
		if err != nil {
			return err
		}
		logrus.Debugf("Cached plugin metadata in %v", c.path)
	}

	// Store in in-memory
	c.inMem = expandAliases(mds)
	c.lastUpdated = time.Now()
	return nil
}

// Registry is a remote list of available plugins
type Registry string

func (r Registry) FetchAll() (map[string]*Metadata, error) {
	var registry map[string]*Metadata
	var bs []byte
	var err error
	if strings.HasPrefix(string(r), "/") {
		bs, err = ioutil.ReadFile(string(r))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch filesystem registry %v: %v", r, err)
		}
	} else {
		resp, err := http.Get(string(r))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch HTTP registry %v: %v", r, err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to fetch HTTP registry %v: %v", r, resp.Status)
		}
		bs, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read HTTP registry %v: %v", r, resp.Status)
		}
		resp.Body.Close()
	}
	err = yaml.Unmarshal(bs, &registry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry %v: %v", r, err)
	}
	return registry, nil
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
			if existing, ok := mds[alias]; ok {
				logrus.Debugf("Alias conflict for %v: used by %v and %v (ignored)",
					alias, existing.Name, md.Name)
				continue
			}
			entries[alias] = md
		}
	}
	return entries
}
