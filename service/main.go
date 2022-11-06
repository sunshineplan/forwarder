package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunshineplan/forwarder"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/service"
	"github.com/sunshineplan/utils/mail"
	"github.com/vharitonsky/iniflags"
)

var svc = service.Service{
	Name:     "MailForward",
	Desc:     "Mail Forward Service",
	Exec:     run,
	TestExec: test,
	Options: service.Options{
		Dependencies: []string{"Wants=network-online.target", "After=network.target"},
		UpdateURL:    "https://github.com/sunshineplan/forwarder/releases/latest/download/forwarder",
	},
}

var (
	accounts = flag.String("accounts", "", "Path to account list file")
	current  = flag.String("current", "", "Path to current map file")
	logPath  = flag.String("log", "", "Path to log file")
	interval = flag.Duration("interval", forwarder.DefaultInterval, "Default refresh interval")
	key      = flag.String("key", "forwarder", "Encrypt key")
)

var emptyDialer mail.Dialer
var defaultSender = new(mail.Dialer)

func main() {
	self, err := os.Executable()
	if err != nil {
		log.Fatalln("Failed to get self path:", err)
	}
	flag.StringVar(&defaultSender.Password, "password", "", "Mail account password")
	flag.StringVar(&svc.Options.UpdateURL, "update", svc.Options.UpdateURL, "Update URL")
	iniflags.SetConfigFile(filepath.Join(filepath.Dir(self), "config.ini"))
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.SetAllowUnknownFlags(true)
	parse()

	if defaultSender.Password != "" {
		password, err := cipher.DecryptText(*key, defaultSender.Password)
		if err != nil {
			log.Println("Failed to decrypt mail account password:", err)
		} else {
			defaultSender.Password = password
		}
	}
	if *defaultSender != emptyDialer && defaultSender.Port == 0 {
		defaultSender.Port = 587
	}

	if *accounts == "" {
		*accounts = filepath.Join(filepath.Dir(self), "accounts.json")
	}
	if *current == "" {
		*current = filepath.Join(filepath.Dir(self), "current.txt")
	}
	if *logPath == "" {
		*logPath = filepath.Join(filepath.Dir(self), "forwarder.log")
	}
	if *interval != 0 {
		forwarder.DefaultInterval = *interval
	}

	if service.IsWindowsService() {
		svc.Run(false)
		return
	}

	switch flag.NArg() {
	case 0:
		run()
	case 1:
		switch flag.Arg(0) {
		case "run":
			svc.Run(false)
		case "debug":
			svc.Run(true)
		case "test":
			err = svc.Test()
		case "install":
			err = svc.Install()
		case "remove":
			err = svc.Remove()
		case "start":
			err = svc.Start()
		case "stop":
			err = svc.Stop()
		case "restart":
			err = svc.Restart()
		case "update":
			err = svc.Update()
		default:
			log.Fatalln("Unknown argument:", flag.Arg(0))
		}
	default:
		log.Fatalln("Unknown arguments:", strings.Join(flag.Args(), " "))
	}
	if err != nil {
		log.Fatalf("Failed to %s: %v", flag.Arg(0), err)
	}
}
