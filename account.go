package forwarder

import (
	"context"
	"fmt"
	"log"
	mailpkg "net/mail"
	"strconv"
	"time"

	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
	"golang.org/x/net/publicsuffix"
)

var DefaultInterval = time.Minute

type Account struct {
	Server   string
	Port     int
	IsTLS    bool `json:"tls"`
	Username string
	Password string

	Current int
	Running bool

	Sender *mail.Dialer

	To mail.Receipts

	Keep bool

	Refresh string
}

func (a Account) domain() string {
	domain, err := publicsuffix.EffectiveTLDPlusOne(a.Server)
	if err != nil {
		panic(err)
	}

	return domain
}

func (a Account) client() (*pop3.Client, error) {
	dial := pop3.Dial
	if a.IsTLS {
		dial = pop3.DialTLS
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := dial(ctx, fmt.Sprintf("%s:%d", a.Server, a.Port))
	if err != nil {
		return nil, err
	}

	if err := ntlmAuth(client, a.domain(), a.Username, a.Password); err != nil {
		return nil, err
	}

	return client, nil
}

func (a Account) Address() string {
	if addr, err := mailpkg.ParseAddress(a.Username); err == nil {
		return addr.Address
	} else {
		return fmt.Sprintf("%s@%s", a.Username, a.domain())
	}
}

func (a *Account) Start(dryRun bool) (res Result, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	return (&forwarder{a}).run(a, dryRun)
}

func (a *Account) Run(success chan<- int, cancel <-chan struct{}) {
	if _, err := strconv.Atoi(a.Refresh); err == nil {
		a.Refresh += "s"
	}
	refresh, err := time.ParseDuration(a.Refresh)
	if err != nil {
		log.Print(err)
		refresh = DefaultInterval
	} else if refresh == 0 {
		refresh = DefaultInterval
	}

	t := time.NewTicker(refresh)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if a.Running {
				log.Printf("%s - [WARN]Previous operation has not finished.", a.Address())
				break
			} else {
				a.Running = true
			}

			if res, err := a.Start(false); err != nil {
				log.Printf("%s - [ERROR]%s", a.Address(), err)
			} else {
				if res.Success+res.Failure > 0 {
					log.Printf("%s - success: %d, failure: %d", a.Address(), res.Success, res.Failure)
					if res.Last != 0 {
						a.Current = res.Last
					}
					if success != nil {
						success <- a.Current
					}
				}
			}

			a.Running = false

		case <-cancel:
			return
		}
	}
}
