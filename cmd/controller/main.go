package main

import (
	"flag"
	"fmt"
	"os"
	"time"

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
	for i := 0; i < 100; i++ {
		fmt.Println(i)
		time.Sleep(time.Second)
	}
}
