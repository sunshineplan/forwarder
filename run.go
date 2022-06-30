package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/sunshineplan/utils/mail"
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
		log.Println("failed to init account list:", err)
	}

	accountWathcer, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
	} else {
		defer accountWathcer.Close()

		if err = accountWathcer.Add(*accounts); err != nil {
			log.Println("failed to watching account list file:", err)
		}
	}
	if err != nil && len(accountList) == 0 {
		return
	}

	if err := loadCurrentMap(); err != nil {
		log.Print(err)
	}

	for {
		c := make(chan struct{})

		accountMutex.Lock()
		l := len(accountList)
		for _, i := range accountList {
			go i.run(c)
		}
		accountMutex.Unlock()

		if accountWathcer != nil {
			if event, ok := <-accountWathcer.Events; ok {
				log.Println("account list file operation:", event.Op)
				switch op := event.Op.String(); op {
				case "CREATE", "WRITE":
					close(c)
					if err := loadAccountList(); err != nil {
						log.Println("failed to load account list:", err)
					}
				case "REMOVE", "RENAME":
					close(c)
					accountMutex.Lock()
					accountList = nil
					accountMutex.Unlock()
				}
				continue
			} else {
				log.Print("account list file watcher closed")
			}
		}

		if l == 0 {
			return
		}
		<-c
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
