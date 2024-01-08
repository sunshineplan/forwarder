package main

import (
	"fmt"
	logpkg "log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sunshineplan/utils/log"
	"github.com/sunshineplan/utils/mail"
)

func run() error {
	svc.Logger = log.New(*logPath, "", log.LstdFlags)
	logpkg.SetOutput(svc.Logger.Writer())

	if err := loadAccountList(); err != nil {
		svc.Println("failed to init account list:", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err = w.Add(filepath.Dir(*accounts)); err != nil {
		return err
	}

	if err := loadCurrent(); err != nil {
		svc.Print(err)
	}

	s, e := make(chan string), make(chan error)
	go func() {
		for {
			select {
			case current := <-s:
				if current != "" {
					go saveCurrent()
				}
			case err := <-e:
				if err != nil {
					go alert(err)
				}
			}
		}
	}()

	for {
		c := make(chan struct{})

		accountMutex.Lock()
		for _, i := range accountList {
			go i.Run(s, e, c)
		}
		accountMutex.Unlock()

	Loop:
		for {
			select {
			case err, ok := <-w.Errors:
				if !ok {
					svc.Println(*accounts, "watcher closed")
				} else {
					svc.Print(err)
				}
				break Loop
			case event, ok := <-w.Events:
				if !ok {
					svc.Println(*accounts, "watcher closed")
					break Loop
				}
				if event.Name == *accounts {
					svc.Println(*accounts, "operation:", event.Op)
					switch {
					case event.Has(fsnotify.Create), event.Has(fsnotify.Write):
						svc.Print("account list file changed")
						close(c)
						if err := loadAccountList(); err != nil {
							svc.Println("failed to load account list:", err)
						}
					case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
						svc.Print("account list file removed")
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

func alert(err error) {
	if admin != nil {
		if err := defaultSender.Send(&mail.Message{
			To:      admin,
			Subject: "[Forwarder]Failed Alert",
			Body:    err.Error(),
		}); err != nil {
			svc.Print(err)
		}
	}
}

func test() error {
	var errCount int
	if err := loadAccountList(); err != nil {
		svc.Println("failed to load account:", err)
		errCount++
	} else {
		for i, account := range accountList {
			address := account.Address()
			if res, err := account.Start(true); err != nil {
				svc.Printf("[%d] %s: %s", i+1, address, err)
				errCount++
			} else if res.Last == "" {
				svc.Printf("[%d] %s has no mails on the server", i+1, address)
			} else {
				svc.Printf("[%d] %s last UID is %s", i+1, address, res.Last)
			}
		}
	}

	if *defaultSender != emptyDialer {
		if err := defaultSender.Send(&mail.Message{
			To:      mail.Receipts{mail.Receipt("", defaultSender.Account)},
			Subject: "Test Mail",
			Body:    "Test",
		}); err != nil {
			svc.Println("failed to send test mail:", err)
			errCount++
		}
	}

	if errCount == 0 {
		return nil
	}
	return fmt.Errorf("%d error(s) encountered", errCount)
}
