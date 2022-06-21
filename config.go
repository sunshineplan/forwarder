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

	"github.com/sunshineplan/utils/txt"
	"github.com/sunshineplan/utils/watcher"
)

const defaultInterval = 5 * time.Minute

var (
	accountList  []account
	accountMutex sync.Mutex

	currentMap   = make(map[string]int)
	currentMutex sync.Mutex

	accountWathcer *watcher.Watcher
)

func loadAccountList() error {
	b, err := os.ReadFile(*accounts)
	if err != nil {
		return err
	}

	accountMutex.Lock()
	defer accountMutex.Unlock()

	return json.Unmarshal(b, &accountList)
}

func loadCurrentMap() error {
	rows, err := txt.ReadFile(*current)
	if err != nil {
		return err
	}

	m := make(map[string]int)
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
		m[strings.TrimSpace(fields[0])] = last
	}

	currentMutex.Lock()
	defer currentMutex.Unlock()

	currentMap = m
	return nil
}

func saveCurrentMap(address string, last int) {
	currentMutex.Lock()
	defer currentMutex.Unlock()

	currentMap[address] = last

	accountMutex.Lock()
	defer accountMutex.Unlock()

	var rows []string
	for _, i := range accountList {
		if current := currentMap[i.address()]; current > 0 {
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
