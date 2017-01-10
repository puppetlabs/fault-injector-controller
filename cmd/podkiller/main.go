package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/podkiller"
	"github.com/puppetlabs/fault-injector-controller/version"
)

var (
	printVersion bool
	printImage   bool
)

func init() {
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.BoolVar(&printVersion, "version", false, "Show version and quit")
	flagset.BoolVar(&printImage, "image", false, "Show the image name for the application and quit")
	flagset.Parse(os.Args[1:])

	rand.Seed(time.Now().UnixNano())
}

func main() {
	if printVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}
	if printImage {
		fmt.Printf("%v/fault-injector-podkiller:%v\n", version.ImageRepo, version.Version)
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
