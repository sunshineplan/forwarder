package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/txt"
)

const defaultInterval = time.Minute

var (
	accountList  []*account
	accountMutex sync.Mutex

	currentMap sync.Map
	operation  sync.Map
)

func loadAccountList() error {
	b, err := os.ReadFile(*accounts)
	if err != nil {
		return err
	}

	accountMutex.Lock()
	defer accountMutex.Unlock()

	accountList = nil
	if err := json.Unmarshal(b, &accountList); err != nil {
		return err
	}

	for _, i := range accountList {
		if i.Port == 0 {
			if i.IsTLS {
				i.Port = 995
			} else {
				i.Port = 110
			}
		}

		if i.Sender != nil {
			if i.Sender.Port == 0 {
				i.Sender.Port = 587
			}

			password, err := cipher.DecryptText(*key, i.Sender.Password)
			if err != nil {
				log.Printf("%s - [WARN]Failed to decrypt sender password: %s", i.address(), err)
			} else {
				i.Sender.Password = password
			}
		}
	}
	log.Printf("loaded %d account(s)", len(accountList))

	return nil
}

func loadCurrentMap() error {
	rows, err := txt.ReadFile(*current)
	if err != nil {
		return err
	}

	for _, row := range rows {
		fields := strings.FieldsFunc(row, func(c rune) bool { return c == ':' })
		if l := len(fields); l == 0 {
			continue
		} else if l != 2 {
			log.Println("invalid value:", row)
			continue
		}
		last, err := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err != nil {
			log.Println("invalid value:", row)
			continue
		}
		currentMap.Store(strings.TrimSpace(fields[0]), last)
	}

	return nil
}

func saveCurrentMap() {
	accountMutex.Lock()
	defer accountMutex.Unlock()

	var rows []string
	for _, i := range accountList {
		if current, ok := currentMap.Load(i.address()); ok && current.(int) > 0 {
			rows = append(rows, fmt.Sprintf("%s:%d", i.address(), current))
		}
	}
	if len(rows) > 0 {
		err := txt.ExportFile(rows, *current)
		if err != nil {
			log.Print(err)
		}
	}
}
