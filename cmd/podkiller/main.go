package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/puppetlabs/fault-injector-controller/pkg/podkiller"
	"github.com/puppetlabs/fault-injector-controller/version"

	"k8s.io/client-go/1.5/pkg/api"
)

var (
	cfg          podkiller.Config
	printVersion bool
	printImage   bool
)

func init() {
	var namespaceValue string
	var namespaceFile string
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagset.StringVar(&namespaceValue, "namespace", "", "The namespace to work in. Mutually exclusive with -namespace-file.")
	flagset.StringVar(&namespaceFile, "namespace-file", "", "A file containing the namespace to work in. Mutually exclusive with -namespace.")
	flagset.StringVar(&cfg.Host, "apiserver", "", "API Server addr, e.g. ' - NOT RECOMMENDED FOR PRODUCTION - http://127.0.0.1:8080'. Omit parameter to run in on-cluster mode and utilize the service account token.")
	flagset.StringVar(&cfg.TLSConfig.CertFile, "cert-file", "", " - NOT RECOMMENDED FOR PRODUCTION - Path to public TLS certificate file.")
	flagset.StringVar(&cfg.TLSConfig.KeyFile, "key-file", "", "- NOT RECOMMENDED FOR PRODUCTION - Path to private TLS certificate file.")
	flagset.StringVar(&cfg.TLSConfig.CAFile, "ca-file", "", "- NOT RECOMMENDED FOR PRODUCTION - Path to TLS CA file.")
	flagset.BoolVar(&cfg.TLSInsecure, "tls-insecure", false, "- NOT RECOMMENDED FOR PRODUCTION - Don't verify API server's CA certificate.")
	flagset.BoolVar(&printVersion, "version", false, "Show version and quit")
	flagset.BoolVar(&printImage, "image", false, "Show the image name for the application and quit")
	flagset.Parse(os.Args[1:])

	if namespaceValue != "" && namespaceFile != "" {
		fmt.Fprint(os.Stderr, "Cannot specify both -namespace and -namespace-file!")
		os.Exit(1)
	}

	// Pick whichever of namespaceValue or namespaceFile is set.
	if namespaceValue != "" {
		cfg.Namespace = namespaceValue
	} else if namespaceFile != "" {
		rawString, err := ioutil.ReadFile(namespaceFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when attempting to read namespace from %v: %v", namespaceFile, err)
			os.Exit(1)
		}
		cfg.Namespace = strings.TrimSpace(string(rawString))
	} else {
		cfg.Namespace = api.NamespaceDefault
	}

	rand.Seed(time.Now().UnixNano())
}

func main() {
	if printVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}
	if printImage {
		fmt.Printf("%v/fault-injector-podkiller:%v\n", version.ImageRepo, version.Version)
		os.Exit(0)
	}
	fmt.Printf("FaultInjector PodKiller, version %v\n", version.Version)
	p, err := podkiller.New(cfg)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.Run(time.Minute, make(chan struct{})); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
