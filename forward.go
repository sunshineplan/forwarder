package forwarder

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"strconv"

	"github.com/Azure/go-ntlmssp"
	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
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
	Success int
	Failure int
}

func (f *forwarder) run(account *Account, dryRun bool) (res Result, err error) {
	msgs, err := f.Uidl(0)
	if err != nil {
		return
	}

	var success, failure int
	for _, msg := range msgs {
		var n int
		n, err = strconv.Atoi(msg.UID)
		if err != nil {
			return
		}

		if account.Current > 0 && account.Current >= n {
			continue
		}

		if dryRun {
			success++
			account.Current = n
		} else {
			if forwardErr := f.forward(account.Sender, msg.ID, account.To, !account.Keep); forwardErr != nil {
				failure++
				log.Print(forwardErr)
			} else {
				success++
				if account.Keep {
					account.Current = n
				}
			}
		}
	}
	res = Result{account.Current, success, failure}

	return
}
