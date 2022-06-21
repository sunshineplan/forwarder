package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	},
}

var (
	accounts = flag.String("accounts", "", "Path to account list file")
	current  = flag.String("current", "", "Path to current map file")
	key      = flag.String("key", "forwarder", "Encrypt key")
	interval = flag.Duration("interval", defaultInterval, "Default refresh interval")
	logPath  = flag.String("log", "", "Log path")
)

var dialer = new(mail.Dialer)

func main() {
	self, err := os.Executable()
	if err != nil {
		log.Fatalln("Failed to get self path:", err)
	}

	flag.StringVar(&dialer.Server, "server", "", "Mail server address")
	flag.IntVar(&dialer.Port, "port", 587, "Mail server port")
	flag.StringVar(&dialer.Account, "account", "", "Mail account name")
	flag.StringVar(&dialer.Password, "password", "", "Mail account password")
	iniflags.SetConfigFile(filepath.Join(filepath.Dir(self), "config.ini"))
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()

	if dialer.Password == "" {
		log.Fatal("Mail account password is empty.")
	}
	dialer.Password, err = cipher.DecryptText(*key, dialer.Password)
	if err != nil {
		log.Fatalf("Failed to decrypt mail account password: %s", err)
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
