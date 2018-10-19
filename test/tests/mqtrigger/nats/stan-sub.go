// This file originally came from official Nats.io GitHub repository.
// You can reach original file with the following link:
// https://github.com/nats-io/go-nats-streaming/tree/master/examples

// Copyright 2012-2016 Apcera Inc. All rights reserved.
// +build ignore

package main

import (
	"flag"
	"log"
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/go-nats-streaming/pb"
)

var usageStr = `
Usage: stan-sub [options] <subject>

Options:
	-s, --server   <url>            NATS Streaming server URL(s)
	-c, --cluster  <cluster name>   NATS Streaming cluster name
	-id,--clientid <client ID>      NATS Streaming client ID

Subscription Options:
	--qgroup <name>                 Queue group
	--seq <seqno>                   Start at seqno
	--all                           Deliver all available messages
	--last                          Deliver starting with last published message
	--since <duration>              Deliver messages in last interval (e.g. 1s, 1hr)
	         (for more information: https://golang.org/pkg/time/#ParseDuration)
	--durable <name>                Durable subscriber name
	--unsubscribe                   Unsubscribe the durable on exit
`

// NOTE: Use tls scheme for TLS, e.g. stan-sub -s tls://demo.nats.io:4443 foo
func usage() {
	log.Fatalf(usageStr)
}

func printMsg(m *stan.Msg) {
	log.Printf("[%s]: '%s'", m.Subject, m.Data)
}

func main() {
	var clusterID string
	var clientID string
	var showTime bool
	var startSeq uint64
	var startDelta string
	var deliverAll bool
	var deliverLast bool
	var durable string
	var qgroup string
	var unsubscribe bool
	var URL string

	//	defaultID := fmt.Sprintf("client.%s", nuid.Next())

	flag.StringVar(&URL, "s", stan.DefaultNatsURL, "The nats server URLs (separated by comma)")
	flag.StringVar(&URL, "server", stan.DefaultNatsURL, "The nats server URLs (separated by comma)")
	flag.StringVar(&clusterID, "c", "test-cluster", "The NATS Streaming cluster ID")
	flag.StringVar(&clusterID, "cluster", "test-cluster", "The NATS Streaming cluster ID")
	flag.StringVar(&clientID, "id", "", "The NATS Streaming client ID to connect with")
	flag.StringVar(&clientID, "clientid", "", "The NATS Streaming client ID to connect with")
	flag.BoolVar(&showTime, "t", false, "Display timestamps")
	// Subscription options
	flag.Uint64Var(&startSeq, "seq", 0, "Start at sequence no.")
	flag.BoolVar(&deliverAll, "all", false, "Deliver all")
	flag.BoolVar(&deliverLast, "last", false, "Start with last value")
	flag.StringVar(&startDelta, "since", "", "Deliver messages since specified time offset")
	flag.StringVar(&durable, "durable", "", "Durable subscriber name")
	flag.StringVar(&qgroup, "qgroup", "", "Queue group name")
	flag.BoolVar(&unsubscribe, "unsubscribe", false, "Unsubscribe the durable on exit")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if clientID == "" {
		log.Printf("Error: A unique client ID must be specified.")
		usage()
	}
	if len(args) < 1 {
		log.Printf("Error: A subject must be specified.")
		usage()
	}

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(URL))
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err, URL)
	}
	// log.Printf("Connected to %s clusterID: [%s] clientID: [%s]\n", URL, clusterID, clientID)

	subj := args[0]

	exit := make(chan struct{})
	mcb := func(msg *stan.Msg) {
		printMsg(msg)
		exit <- struct{}{}
	}

	startOpt := stan.StartAt(pb.StartPosition_NewOnly)

	if startSeq != 0 {
		startOpt = stan.StartAtSequence(startSeq)
	} else if deliverLast == true {
		startOpt = stan.StartWithLastReceived()
	} else if deliverAll == true {
		log.Print("subscribing with DeliverAllAvailable")
		startOpt = stan.DeliverAllAvailable()
	} else if startDelta != "" {
		ago, err := time.ParseDuration(startDelta)
		if err != nil {
			sc.Close()
			log.Fatal(err)
		}
		startOpt = stan.StartAtTimeDelta(ago)
	}

	sub, err := sc.QueueSubscribe(subj, qgroup, mcb, startOpt, stan.DurableName(durable))
	if err != nil {
		sc.Close()
		log.Fatal(err)
	}

	<-exit
	sub.Unsubscribe()
}
