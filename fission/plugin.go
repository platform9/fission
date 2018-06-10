package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fission/fission/fission/plugins"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var cmdPlugin = cli.Command{
	Name:    "plugin",
	Aliases: []string{"plugins"},
	Usage:   "Manage Fission CLI plugins",
	Subcommands: []cli.Command{
		{
			Name:   "list",
			Usage:  "List installed plugins",
			Action: pluginList,
		},
		{
			Name:   "cache-clear",
			Usage:  "Clear plugin caches",
			Action: pluginCacheClear,
		},
	},
}

func pluginList(c *cli.Context) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tPATH")
	for _, plugin := range plugins.FindAll() {
		fmt.Fprintf(w, "%v\t%v\t%v\n", plugin.Name, plugin.Version, plugin.Path)
	}
	w.Flush()
	return nil
}

func pluginCacheClear(c *cli.Context) error {
	plugins.ClearCache()
	logrus.Debug("Cache cleared.")
	return nil
}
