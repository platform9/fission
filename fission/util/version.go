package util

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/fission/fission"
	"github.com/fission/fission/controller/client"
	"github.com/fission/fission/fission/log"
	"github.com/fission/fission/fission/plugin"
)

// Versions is a container of versions of the client (and its plugins) and server (and its plugins).
type Versions struct {
	Client map[string]fission.BuildMeta `json:"client"`
	Server map[string]fission.BuildMeta `json:"server"`
}

func GetVersion(client *client.Client) []byte {
	serverInfo, err := client.ServerInfo()
	if err != nil {
		log.Warn(fmt.Sprintf("Error getting Fission API version: %v", err))
	}

	// Fetch client versions
	versions := Versions{
		Client: map[string]fission.BuildMeta{
			"fission/core": fission.BuildInfo(),
		},
	}
	for _, pmd := range plugin.FindAll() {
		versions.Client[pmd.Name] = fission.BuildMeta{
			Version: pmd.Version,
		}
	}

	// Fetch server versions
	versions.Server = map[string]fission.BuildMeta{
		"fission/core": serverInfo.Build,
	}
	// FUTURE: fetch versions of plugins server-side
	bs, err := yaml.Marshal(versions)
	if err != nil {
		log.Fatal("Failed to format versions: " + err.Error())
	}

	return bs
}
