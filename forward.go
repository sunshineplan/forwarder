package forwarder

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/Azure/go-ntlmssp"
	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
	"github.com/sunshineplan/utils/workers"
)

var (
	emptyDialer    mail.Dialer
	errEmptyDialer = errors.New("empty dialer configuration")
)

func ntlmAuth(client *pop3.Client, domain, username, password string) (err error) {
	if _, err = client.Cmd("AUTH NTLM", false); err != nil {
		return
	}

	b, err := ntlmssp.NewNegotiateMessage(domain, "")
	if err != nil {
		return
	}

	s, err := client.Cmd(base64.StdEncoding.EncodeToString(b), false)
	if err != nil {
		return
	}

	b, err = base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	b, err = ntlmssp.ProcessChallenge(b, username, password, false)
	if err != nil {
		return
	}

	_, err = client.Cmd(base64.StdEncoding.EncodeToString(b), false)

	return
}

type forwarder struct {
	*pop3.Client
	authFunc func(client *pop3.Client, domain, username, password string) error
}

func (f *forwarder) auth(domain, username, password string) error {
	if f.authFunc == nil {
		f.authFunc = ntlmAuth
	}
	return f.authFunc(f.Client, domain, username, password)
}

func (f *forwarder) forward(sender *mail.Dialer, id int, to []string, delete bool) error {
	if sender == nil || *sender == emptyDialer {
		return errEmptyDialer
	}

	s, err := f.Retr(id)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultInterval)
	defer cancel()

	if err := sender.SendMail(ctx, sender.Account, to, []byte(s)); err != nil {
		return err
	}

	if delete {
		return f.Dele(id)
	} else {
		return nil
	}
}

type Result struct {
	Last    int
	Success int64
	Failure int64
}

func (f *forwarder) run(account *Account, dryRun bool) (res Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = fmt.Errorf("unknown panic: %v", x)
			}
		}
	}()

	msgs, err := f.Uidl(0)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	var mu sync.Mutex
	var index int
	var success, failure atomic.Int64
	current := account.Current
	workers.Slice(msgs, func(i int, msg pop3.MessageID) {
		if ctx.Err() != nil {
			return
		}

		n, err := strconv.Atoi(msg.UID)
		if err != nil {
			cancel(err)
			return
		}

		if current > 0 && current >= n {
			return
		}

		if dryRun {
			success.Add(1)
			mu.Lock()
			if i >= index {
				index = i
				account.Current = n
			}
			mu.Unlock()
		} else {
			if forwardErr := f.forward(account.Sender, msg.ID, account.To, !account.Keep); forwardErr != nil {
				failure.Add(1)
				log.Print(forwardErr)
			} else {
				mu.Lock()
				success.Add(1)
				if account.Keep && i >= index {
					index = i
					account.Current = n
				}
				mu.Unlock()
			}
		}
	})
	if err = context.Cause(ctx); err != nil {
		return
	}
	res = Result{account.Current, success.Load(), failure.Load()}

	return
}
