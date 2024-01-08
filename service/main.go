package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/sunshineplan/forwarder"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/service"
	"github.com/sunshineplan/utils/flags"
	"github.com/sunshineplan/utils/mail"
)

var (
	svc = service.New()

	emptyDialer   mail.Dialer
	defaultSender = new(mail.Dialer)
)

func init() {
	svc.Name = "MailForward"
	svc.Desc = "Mail Forward Service"
	svc.Exec = run
	svc.TestExec = test
	svc.Options = service.Options{
		Dependencies: []string{"Wants=network-online.target", "After=network.target"},
		UpdateURL:    "https://github.com/sunshineplan/forwarder/releases/latest/download/forwarder",
	}
}

var (
	accounts = flag.String("accounts", "", "Path to account list file")
	current  = flag.String("current", "", "Path to current map file")
	logPath  = flag.String("log", "", "Path to log file")
	interval = flag.Duration("interval", forwarder.DefaultInterval, "Default refresh interval")
	key      = flag.String("key", "forwarder", "Encrypt key")
	admin    mail.Receipts
)

func main() {
	self, err := os.Executable()
	if err != nil {
		svc.Fatalln("Failed to get self path:", err)
	}
	flag.TextVar(&admin, "admin", mail.Receipts(nil), "Admin mail")
	flag.StringVar(&defaultSender.Password, "password", "", "Mail account password")
	flag.StringVar(&svc.Options.UpdateURL, "update", svc.Options.UpdateURL, "Update URL")
	flags.SetConfigFile(filepath.Join(filepath.Dir(self), "config.ini"))
	parse()

	if defaultSender.Password != "" {
		password, err := cipher.DecryptText(*key, defaultSender.Password)
		if err != nil {
			svc.Println("Failed to decrypt mail account password:", err)
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

	if err := svc.ParseAndRun(flag.Args()); err != nil {
		svc.Fatal(err)
	}
}
