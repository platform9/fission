package plugin

// builtinRegistry consists of a map of plugin names along with the relevant url.
var builtinRegistry = map[string]string{
	"workflows": "https://github.com/fission/fission-workflows/releases",
	"foo":       "https://github.com/fission/fission-workflows/releases",
}

// SearchRegistries will search (remote) registries for the presence of the command.
// For now no
func SearchRegistries(cmd string) (string, bool) {
	url, ok := builtinRegistry[cmd]
	return url, ok
}
