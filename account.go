package main

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"sync"
	"time"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/pop3"
	"golang.org/x/net/publicsuffix"
)

type account struct {
	Server   string
	Port     int
	IsTLS    bool `json:"tls"`
	Username string
	Password string

	To []string

	Keep bool

	Refresh string
}

func (a account) domain() string {
	domain, err := publicsuffix.EffectiveTLDPlusOne(a.Server)
	if err != nil {
		panic(err)
	}

	return domain
}

func (a account) address() string {
	if addr, err := mail.ParseAddress(a.Username); err == nil {
		return addr.Address
	} else {
		return fmt.Sprintf("%s@%s", a.Username, a.domain())
	}
}

func (a account) connect() (*pop3.Client, error) {
	var dial func(context.Context, string) (*pop3.Client, error)
	if a.IsTLS {
		if a.Port == 0 {
			a.Port = 995
		}
		dial = pop3.DialTLS
	} else {
		if a.Port == 0 {
			a.Port = 110
		}
		dial = pop3.Dial
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return dial(ctx, fmt.Sprintf("%s:%d", a.Server, a.Port))
}

func (a account) start() (res result, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	password, err := cipher.DecryptText(*key, a.Password)
	if err != nil {
		return
	}

	client, err := a.connect()
	if err != nil {
		return
	}
	defer client.Quit()

	f := &forwarder{client, nil}
	if err = f.auth(a.domain(), a.Username, password); err != nil {
		return
	}

	currentMutex.Lock()
	current := currentMap[a.address()]
	currentMutex.Unlock()

	return f.run(current, a.To, !a.Keep)
}

func (a account) run(cancel <-chan struct{}) {
	refresh, err := time.ParseDuration(a.Refresh)
	if a.Refresh != "" && err != nil {
		log.Printf("%s - [ERROR]: %s", a.address(), err)
	}
	if refresh == 0 {
		refresh = *interval
	}

	t := time.NewTicker(refresh)
	defer t.Stop()

	var locker sync.Mutex
	for {
		select {
		case <-t.C:
			if !locker.TryLock() {
				log.Printf("%s - [WARN]: Previous operation has not finished.", a.address())
				break
			}

			if res, err := a.start(); err != nil {
				log.Printf("%s - [ERROR]: %s", a.address(), err)
			} else {
				if res.success+res.failure > 0 {
					log.Printf("%s - success: %d, failure: %d", a.address(), res.success, res.failure)
					if res.last != 0 {
						saveCurrentMap(a.address(), res.last)
					}
				}
			}

			locker.Unlock()

		case <-cancel:
			return
		}
	}
}
