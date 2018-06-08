// Package plugin provides support for creating extensible CLIs
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	CmdTimeout      = 5 * time.Second
	CmdMetadataArgs = "--plugin"
	PrefixFission   = "fission-"
)

var (
	ErrPluginNotFound = errors.New("plugin not found")
	ErrPluginInvalid  = errors.New("invalid plugin")
)

// Metadata contains the metadata of a plugin.
// The only metadata that is guaranteed to be non-empty is the Path and Name. All other fields are considered optional.
type Metadata struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Url      string            `json:"url"`
	Requires map[string]string `json:"requires"`
	Aliases  []string          `json:"aliases"`
	Usage    string            `json:"usage"`
	Path     string            `json:"path"`
}

// Find searches the machine for the given plugin, returning the metadata of the plugin.
// The only metadata that is guaranteed to be non-empty is the Path and Name. All other fields are considered optional.
// If found it returns the plugin, otherwise it returns ErrPluginNotFound if the plugin was not found or it returns
// ErrPluginInvalid if the plugin was found but considered unusable (e.g. not executable or invalid permissions).
func Find(pluginName string) (*Metadata, error) {
	// look in cache
	// else, look in path
	// return findPlugin(pluginName)
	pluginPath, err := findPluginPath(pluginName)
	if err != nil {
		return nil, err
	}

	md, err := fetchPluginMetadata(pluginPath)
	if err != nil {
		return nil, err
	}
	return md, nil
}

// Exec executes the plugin using the provided args.
// All input and output is redirected to stdin, stdout, and stderr.
func Exec(pluginMetadata *Metadata, args []string) error {
	cmd := exec.Command(pluginMetadata.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FindAll searches the machine for all plugins currently present.
func FindAll(pluginPrefix string) map[string]*Metadata {
	plugins := map[string]*Metadata{}

	dirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range dirs {
		fs, err := ioutil.ReadDir(dir)
		if err != nil {
			logrus.Debugf("Failed to read $PATH directory: %v", dir)
			continue
		}
		for _, f := range fs {
			if !strings.HasPrefix(f.Name(), pluginPrefix) {
				continue
			}
			fp := path.Join(dir, f.Name())
			md, err := fetchPluginMetadata(fp)
			if err != nil {
				logrus.Debugf("%v: %v", f.Name(), err)
				continue
			}
			// TODO merge aliases
			plugins[md.Name] = md
		}
	}
	return plugins
}

func findPluginPath(pluginName string) (path string, err error) {
	binaryName := binaryNameForPlugin(pluginName)
	path, err = exec.LookPath(binaryName)
	if err != nil {
		logrus.Debugf("Plugin not found on PATH: %v", err)
	}

	if len(path) == 0 {
		return "", ErrPluginNotFound
	}
	return path, nil
}

func fetchPluginMetadata(pluginPath string) (*Metadata, error) {
	buf := bytes.NewBuffer(nil)
	ctx, cancel := context.WithTimeout(context.Background(), CmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pluginPath, CmdMetadataArgs) // Note: issue can occur with signal propagation
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	// Parse metadata if possible
	p := &Metadata{}
	err = json.Unmarshal(buf.Bytes(), p)
	if err != nil {
		logrus.Debugf("Failed to read plugin metadata: %v", err)
		p.Path = pluginPath
		p.Name = pluginNameFromBinary(path.Base(pluginPath))
	}
	return p, nil
}

func binaryNameForPlugin(pluginName string) string {
	return PrefixFission + pluginName
}

func pluginNameFromBinary(binaryName string) string {
	return strings.TrimPrefix(binaryName, PrefixFission)
}
