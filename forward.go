package forwarder

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
)

var (
	emptyDialer    mail.Dialer
	errEmptyDialer = errors.New("empty dialer configuration")
)

type forwarder struct {
	*Account
	*pop3.Client
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
	Last    string
	Success int
	Failure int
}

func (f *forwarder) run(dryRun bool) (res Result, err error) {
	defer func() {
		f.Quit()
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

	count, _, err := f.Stat()
	if err != nil {
		return
	}
	if count == 0 {
		return
	}
	last, err := f.Uidl(count)
	if err != nil {
		return
	}
	if dryRun {
		res.Last = last[0].UID
		return
	}
	var start int
	m := make(map[int]string)
	if f.Keep {
		if f.Current == last[0].UID {
			return
		}
		for i := count; i > 0; i-- {
			var id []pop3.MessageID
			id, err = f.Uidl(i)
			if err != nil {
				return
			}
			m[i] = id[0].UID
			if id[0].UID == f.Current {
				break
			}
			start = i
		}
		if start == 0 {
			return
		}
	} else {
		start = 1
	}

	var success, failure int
	for i := start; i <= count; i++ {
		if err := f.forward(f.Sender, i, f.To.List(), !f.Keep); err != nil {
			failure++
			log.Print(err)
		} else {
			success++
			if f.Keep {
				f.Current = m[i]
			}
		}
	}
	res = Result{f.Current, success, failure}
	return
}
