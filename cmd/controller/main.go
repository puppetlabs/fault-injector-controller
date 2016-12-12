package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/puppetlabs/fault-injector-controller/pkg/controller"
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
		fmt.Println("fault-injector-controller", version.Version)
		os.Exit(0)
	}
	c, err := controller.New()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Run(make(chan struct{})); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
