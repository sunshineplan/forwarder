package main

import (
	"context"
	"fmt"
	"net/mail"
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
