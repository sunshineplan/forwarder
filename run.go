package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/workers"
)

func run() {
	f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open log file:", err)
	} else {
		log.SetOutput(f)
	}

	initAccountList()
	if err := loadCurrentMap(); err != nil {
		log.Print(err)
	}

	var locker sync.Mutex
	for {
		<-time.After(time.Duration(*interval) * time.Minute)

		if !locker.TryLock() {
			log.Print("Previous operation has not finished.")
			continue
		}

		accountMutex.Lock()
		list := make([]account, len(accountList))
		copy(list, accountList)
		accountMutex.Unlock()

		workers.Slice(list, func(_ int, account account) {
			if res, err := account.start(); err != nil {
				log.Print(err)
			} else {
				if res.success+res.failure > 0 {
					log.Printf("%s - success: %d, failure: %d", account.address(), res.success, res.failure)
					if res.last != 0 {
						currentMutex.Lock()
						currentMap[account.address()] = res.last
						currentMutex.Unlock()
					}
				}
			}
		})
		saveCurrentMap()

		locker.Unlock()
	}
}

func test() error {
	if err := loadAccountList(); err != nil {
		return fmt.Errorf("failed to load account: %s", err)
	}
	if err := dialer.Send(&mail.Message{
		To:      []string{dialer.Account},
		Subject: "Test Mail",
		Body:    "Test",
	}); err != nil {
		return fmt.Errorf("failed to send test mail: %s", err)
	}
	return nil
}
