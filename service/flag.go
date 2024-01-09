//go:build !access

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/flags"
	"github.com/sunshineplan/utils/mail"
)

func parse() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:", os.Args[0])
		fmt.Println(`
  -server string
        Mail server address
  -port int
        Mail server port
  -account string
        Mail account name
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
	flag.StringVar(&defaultSender.Server, "server", "", "Mail server address")
	flag.IntVar(&defaultSender.Port, "port", 0, "Mail server port")
	flag.StringVar(&defaultSender.Account, "account", "", "Mail account name")
	flag.StringVar(&defaultSender.Password, "password", "", "Mail account password")
	flag.TextVar(&admin, "admin", mail.Receipts(nil), "Admin mail")
	flags.Parse()

	if defaultSender.Password != "" {
		password, err := cipher.DecryptText(*key, defaultSender.Password)
		if err != nil {
			svc.Println("Failed to decrypt mail account password:", err)
		} else {
			defaultSender.Password = password
		}
	}
}
