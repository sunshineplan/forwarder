package main

import (
	"context"
	"fmt"
	"log"
	m "net/mail"
	"time"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
	"golang.org/x/net/publicsuffix"
)

type account struct {
	Server   string
	Port     int
	IsTLS    bool `json:"tls"`
	Username string
	Password string

	Sender *mail.Dialer

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
	if addr, err := m.ParseAddress(a.Username); err == nil {
		return addr.Address
	} else {
		return fmt.Sprintf("%s@%s", a.Username, a.domain())
	}
}

func (a account) connect() (*pop3.Client, error) {
	dial := pop3.Dial
	if a.IsTLS {
		dial = pop3.DialTLS
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

	var current int
	if v, ok := currentMap.Load(a.address()); ok {
		current = v.(int)
	}

	return f.run(a.Sender, current, a.To, !a.Keep)
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

	for {
		select {
		case <-t.C:
			if _, ok := operation.LoadOrStore(a.address(), nil); ok {
				log.Printf("%s - [WARN]Previous operation has not finished.", a.address())
				break
			}

			if res, err := a.start(); err != nil {
				log.Printf("%s - [ERROR]%s", a.address(), err)
			} else {
				if res.success+res.failure > 0 {
					log.Printf("%s - success: %d, failure: %d", a.address(), res.success, res.failure)
					if res.last != 0 {
						currentMap.Store(a.address(), res.last)
						saveCurrentMap()
					}
				}
			}

			operation.Delete(a.address())

		case <-cancel:
			return
		}
	}
}
