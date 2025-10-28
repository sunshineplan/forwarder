//go:build access

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sunshineplan/utils/flags"
	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/retry"
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
	flags.ParseFlags(false, false)

	var res struct {
		Sender *mail.Dialer
		Admin  mail.Receipts
	}
	if err := retry.Do(func() error {
		resp, err := http.Get((&url.URL{Scheme: "https", Host: *domain, Path: access}).String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(&res)
	}, 5, 60*time.Second); err != nil {
		svc.Fatal("access denied")
	}
	defaultSender = res.Sender
	admin = res.Admin
}
