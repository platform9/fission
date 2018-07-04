// Package plugins provides support for creating extensible CLIs
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
	// Search PATH for plugin as command-name
	// To check if plugin is actually there still.
	pluginPath, err := findPluginOnPath(pluginName)
	if err != nil {
		// Fallback: Search for alias in each command
		mds := FindAll()
		for _, md := range mds {
			if md.HasAlias(pluginName) {
				return md, nil
			}
		}
		return nil, ErrPluginNotFound
	}

	md, err := fetchPluginMetadata(pluginPath)
	if err != nil {
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
	d, err := os.Stat(pluginPath)
	if err != nil {
		return nil, ErrPluginNotFound
	}
	if m := d.Mode(); m.IsDir() || m&0111 == 0 {
		return nil, ErrPluginInvalid
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
	pluginName := strings.TrimPrefix(path.Base(pluginPath), Prefix)
	md := &Metadata{}
	err = json.Unmarshal(buf.Bytes(), md)
	if err != nil {
		md.Name = pluginName
	}
	md.ModifiedAt = d.ModTime()
	md.Path = pluginPath
	md.Alias(pluginName)
	return md, nil
}
