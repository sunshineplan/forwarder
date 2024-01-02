package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sunshineplan/utils/mail"
)

func run() error {
	f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open log file:", err)
	} else {
		log.SetOutput(f)
	}

	if err := loadAccountList(); err != nil {
		log.Println("failed to init account list:", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err = w.Add(filepath.Dir(*accounts)); err != nil {
		return err
	}

	if err := loadCurrent(); err != nil {
		log.Print(err)
	}

	s := make(chan string)
	go func() {
		for current := range s {
			if current != "" {
				saveCurrent()
			}
		}
	}()

	for {
		c := make(chan struct{})

		accountMutex.Lock()
		for _, i := range accountList {
			go i.Run(s, c)
		}
		accountMutex.Unlock()

	Loop:
		for {
			select {
			case err, ok := <-w.Errors:
				if !ok {
					log.Println(*accounts, "watcher closed")
				} else {
					log.Print(err)
				}
				break Loop
			case event, ok := <-w.Events:
				if !ok {
					log.Println(*accounts, "watcher closed")
					break Loop
				}
				if event.Name == *accounts {
					log.Println(*accounts, "operation:", event.Op)
					switch {
					case event.Has(fsnotify.Create), event.Has(fsnotify.Write):
						log.Print("account list file changed")
						close(c)
						if err := loadAccountList(); err != nil {
							log.Println("failed to load account list:", err)
						}
					case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
						log.Print("account list file removed")
						close(c)
						accountMutex.Lock()
						accountList = nil
						accountMutex.Unlock()
					case event.Has(fsnotify.Chmod):
						continue
					}
					break Loop
				}
			}
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
			} else if res.Last == "" {
				log.Printf("[%d] %s has no mails on the server", i+1, address)
			} else {
				log.Printf("[%d] %s last UID is %s", i+1, address, res.Last)
			}
		}
	}

	if *defaultSender != emptyDialer {
		if err := defaultSender.Send(&mail.Message{
			To:      mail.Receipts{mail.Receipt("", defaultSender.Account)},
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
