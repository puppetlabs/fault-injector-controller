package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/puppetlabs/fault-injector-controller/pkg/controller"
	"github.com/puppetlabs/fault-injector-controller/version"
)

var (
	cfg          controller.Config
	printVersion bool
	printImage   bool
)

func init() {
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagset.StringVar(&cfg.Host, "apiserver", "", "API Server addr, e.g. ' - NOT RECOMMENDED FOR PRODUCTION - http://127.0.0.1:8080'. Omit parameter to run in on-cluster mode and utilize the service account token.")
	flagset.StringVar(&cfg.TLSConfig.CertFile, "cert-file", "", " - NOT RECOMMENDED FOR PRODUCTION - Path to public TLS certificate file.")
	flagset.StringVar(&cfg.TLSConfig.KeyFile, "key-file", "", "- NOT RECOMMENDED FOR PRODUCTION - Path to private TLS certificate file.")
	flagset.StringVar(&cfg.TLSConfig.CAFile, "ca-file", "", "- NOT RECOMMENDED FOR PRODUCTION - Path to TLS CA file.")
	flagset.BoolVar(&cfg.TLSInsecure, "tls-insecure", false, "- NOT RECOMMENDED FOR PRODUCTION - Don't verify API server's CA certificate.")
	flagset.BoolVar(&printVersion, "version", false, "Show version and quit")
	flagset.BoolVar(&printImage, "image", false, "Show the image name for the application and quit")
	flagset.Parse(os.Args[1:])
}

func main() {
	if printVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}
	if printImage {
		fmt.Printf("%v/fault-injector-controller:%v\n", version.ImageRepo, version.Version)
	}
	c, err := controller.New(cfg)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Run(make(chan struct{})); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
