package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/watcher"
)

var (
	emptyDialer    mail.Dialer
	emptyDialerErr = errors.New("empty dialer configuration")
)

func run() {
	f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open log file:", err)
	} else {
		log.SetOutput(f)
	}

	if err := loadAccountList(); err != nil {
		log.Fatalln("failed to init account list:", err)
	}
	accountWathcer = watcher.New(*accounts, time.Second)

	if err := loadCurrentMap(); err != nil {
		log.Print(err)
	}

	for {
		c := make(chan struct{})

		accountMutex.Lock()
		for _, i := range accountList {
			go i.run(c)
		}
		accountMutex.Unlock()

		select {
		case <-accountWathcer.C:
			close(c)
			if err := loadAccountList(); err != nil {
				log.Println("failed to load account list:", err)
			}
		}
	}
}

func test() error {
	if err := loadAccountList(); err != nil {
		return fmt.Errorf("failed to load account: %s", err)
	}

	if *defaultSender != emptyDialer {
		if err := defaultSender.Send(&mail.Message{
			To:      []string{defaultSender.Account},
			Subject: "Test Mail",
			Body:    "Test",
		}); err != nil {
			return fmt.Errorf("failed to send test mail: %s", err)
		}
	}

	return nil
}
