package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/sunshineplan/forwarder"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/txt"
)

var (
	accountList  []*forwarder.Account
	accountMutex sync.Mutex
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

		password, err := cipher.DecryptText(*key, i.Password)
		if err != nil {
			log.Print(err)
		} else {
			i.Password = password
		}

		if i.Sender != nil {
			if i.Sender.Port == 0 {
				i.Sender.Port = 587
			}

			password, err := cipher.DecryptText(*key, i.Sender.Password)
			if err != nil {
				log.Printf("%s - [WARN]Failed to decrypt sender password: %s", i.Address(), err)
			} else {
				i.Sender.Password = password
			}
		} else {
			i.Sender = defaultSender
		}
	}
	log.Printf("loaded %d account(s)", len(accountList))

	return nil
}

func loadCurrent() error {
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
		last := strings.TrimSpace(fields[1])
		for _, i := range accountList {
			if address := strings.TrimSpace(fields[0]); i.Address() == address {
				i.Current = last
			}
		}
	}

	return nil
}

func saveCurrent() {
	accountMutex.Lock()
	defer accountMutex.Unlock()

	var rows []string
	for _, i := range accountList {
		if i.Current != "" {
			rows = append(rows, fmt.Sprintf("%s:%s", i.Address(), i.Current))
		}
	}
	if len(rows) > 0 {
		err := txt.ExportFile(rows, *current)
		if err != nil {
			log.Print(err)
		}
	}
}
