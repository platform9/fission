/*
Copyright 2018 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    tttp://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli"

	"github.com/fission/fission/fission/log"
	"github.com/fission/fission/fission/util"
	"github.com/fission/fission/redis/build/gen"
)

func recordsView(c *cli.Context) error {
	var verbosity int
	if c.Bool("v") && c.Bool("vv") {
		log.Fatal("conflicting verbosity levels, use either --v or --vv")
	}
	if c.Bool("v") {
		verbosity = 1
	}
	if c.Bool("vv") {
		verbosity = 2
	}

	function := c.String("function")
	trigger := c.String("trigger")
	from := c.String("from")
	to := c.String("to")

	//Refuse multiple filters for now
	if multipleFiltersSpecified(function, trigger, from+to) {
		log.Fatal("maximum of one filter is currently supported, either --function, --trigger, or --from,--to")
	}

	if len(function) != 0 {
		return recordsByFunction(function, verbosity, c)
	}
	if len(trigger) != 0 {
		return recordsByTrigger(trigger, verbosity, c)
	}
	if len(from) != 0 && len(to) != 0 {
		return recordsByTime(from, to, verbosity, c)
	}
	err := recordsAll(verbosity, c)
	util.CheckErr(err, "view records")
	return nil
}

func recordsAll(verbosity int, c *cli.Context) error {
	fc := util.GetApiClient(c.GlobalString("server"))

	records, err := fc.RecordsAll()
	util.CheckErr(err, "view records")

	showRecords(records, verbosity)

	return nil
}

func recordsByTrigger(trigger string, verbosity int, c *cli.Context) error {
	fc := util.GetApiClient(c.GlobalString("server"))

	records, err := fc.RecordsByTrigger(trigger)
	util.CheckErr(err, "view records")

	showRecords(records, verbosity)

	return nil
}

// TODO: More accurate function name (function filter)
func recordsByFunction(function string, verbosity int, c *cli.Context) error {
	fc := util.GetApiClient(c.GlobalString("server"))

	records, err := fc.RecordsByFunction(function)
	util.CheckErr(err, "view records")

	showRecords(records, verbosity)

	return nil
}

func recordsByTime(from string, to string, verbosity int, c *cli.Context) error {
	fc := util.GetApiClient(c.GlobalString("server"))

	records, err := fc.RecordsByTime(from, to)
	util.CheckErr(err, "view records")

	showRecords(records, verbosity)

	return nil
}

func showRecords(records []*redisCache.RecordedEntry, verbosity int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if verbosity == 1 {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
			"REQUID", "REQUEST METHOD", "FUNCTION", "RESPONSE STATUS", "TRIGGER")
		for _, record := range records {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
				record.ReqUID, record.Req.Method, record.Req.Header["X-Fission-Function-Name"], record.Resp.Status, record.Trigger)
		}
	} else if verbosity == 2 {
		for _, record := range records {
			fmt.Println(record)
		}
	} else {
		fmt.Fprintf(w, "%v\n",
			"REQUID")
		for _, record := range records {
			fmt.Fprintf(w, "%v\n",
				record.ReqUID)
		}
	}
	w.Flush()
}

func multipleFiltersSpecified(entries ...string) bool {
	var specified int
	for _, entry := range entries {
		if len(entry) > 0 {
			specified += 1
		}
	}
	return specified > 1
}
