package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/podkiller"
	"github.com/puppetlabs/fault-injector-controller/version"
)

var (
	printVersion bool
)

func init() {
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.Parse()
}

func main() {
	if printVersion {
		fmt.Println("fault-injector-controller PodKiller", version.Version)
		os.Exit(0)
	}
	p, err := podkiller.New()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.Run(time.Minute, make(chan struct{})); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
