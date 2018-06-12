package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/fission/fission/fission/plugin"
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

func pluginList(_ *cli.Context) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tPATH")
	for _, p := range plugin.FindAll() {
		fmt.Fprintf(w, "%v\t%v\t%v\n", p.Name, p.Version, p.Path)
	}
	w.Flush()
	return nil
}

func pluginCacheClear(_ *cli.Context) error {
	plugin.ClearCache()
	logrus.Debug("Cache cleared.")
	return nil
}
