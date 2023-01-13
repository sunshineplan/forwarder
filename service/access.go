//go:build access

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/sunshineplan/utils/flags"
)

const (
	defaultDomain = ""
	access        = "forwarder"
)

var domain = flag.String("domain", defaultDomain, "Access Domain")

func parse() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:", os.Args[0])
		fmt.Println(`
  -password string
        Mail account password
  -accounts string
        Path to account list file
  -current string
        Path to current map file
  -log string
        Path to log file
  -interval duration
        Default refresh interval (default 1m0s)`)
	}
	flags.Parse()

	url := url.URL{Scheme: "https", Host: *domain, Path: access}
	resp, err := http.Get(url.String())
	if err != nil {
		log.Fatal("access denied")
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&defaultSender); err != nil {
		log.Fatal("access denied")
	}
}
