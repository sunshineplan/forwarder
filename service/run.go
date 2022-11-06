package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/sunshineplan/utils/mail"
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

	s := make(chan int)
	go func() {
		for current := range s {
			if current != 0 {
				saveCurrentMap()
			}
		}
	}()

	for {
		c := make(chan struct{})

		accountMutex.Lock()
		l := len(accountList)
		for _, i := range accountList {
			go i.Run(s, c)
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
	var errCount int
	if err := loadAccountList(); err != nil {
		log.Println("failed to load account:", err)
		errCount++
	} else {
		for i, account := range accountList {
			address := account.Address()
			if res, err := account.Start(true); err != nil {
				log.Printf("[%d] %s: %s", i+1, address, err)
				errCount++
			} else if res.Last == 0 {
				log.Printf("[%d] %s has no mails on the server", i+1, address)
			} else {
				log.Printf("[%d] %s last UID is %d", i+1, address, res.Last)
			}
		}
	}

	if *defaultSender != emptyDialer {
		if err := defaultSender.Send(&mail.Message{
			To:      []string{defaultSender.Account},
			Subject: "Test Mail",
			Body:    "Test",
		}); err != nil {
			log.Println("failed to send test mail:", err)
			errCount++
		}
	}

	if errCount == 0 {
		return nil
	}
	return fmt.Errorf("%d error(s) encountered", errCount)
}
