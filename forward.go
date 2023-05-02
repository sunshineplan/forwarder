package forwarder

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
	"github.com/sunshineplan/utils/workers"
)

var (
	emptyDialer    mail.Dialer
	errEmptyDialer = errors.New("empty dialer configuration")
)

type forwarder struct {
	*Account
}

func (f *forwarder) forward(sender *mail.Dialer, id int, to []string, delete bool) error {
	if sender == nil || *sender == emptyDialer {
		return errEmptyDialer
	}

	client, err := f.client()
	if err != nil {
		return err
	}
	defer client.Quit()

	s, err := client.Retr(id)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultInterval)
	defer cancel()

	if err := sender.SendMail(ctx, sender.Account, to, []byte(s)); err != nil {
		return err
	}

	if delete {
		return client.Dele(id)
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

	var client *pop3.Client
	client, err = f.client()
	if err != nil {
		return
	}
	msgs, err := client.Uidl(0)
	if err != nil {
		return
	}
	client.Quit()

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
			if forwardErr := f.forward(account.Sender, msg.ID, account.To.List(), !account.Keep); forwardErr != nil {
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
