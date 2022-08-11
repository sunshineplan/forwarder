package main

import (
	"context"
	"encoding/base64"
	"log"
	"strconv"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/sunshineplan/utils/executor"
	"github.com/sunshineplan/utils/mail"
	"github.com/sunshineplan/utils/pop3"
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
	if (sender == nil || *sender == emptyDialer) && *defaultSender == emptyDialer {
		return errEmptyDialer
	}

	s, err := f.Retr(id)
	if err != nil {
		return err
	}

	if _, err := executor.ExecuteSerial(
		[]*mail.Dialer{sender, defaultSender},
		func(d *mail.Dialer) (any, error) {
			if d == nil || *d == emptyDialer {
				return nil, executor.SkipErr
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			return nil, d.SendMail(ctx, d.Account, to, []byte(s))
		},
	); err != nil {
		return err
	}

	if delete {
		return f.Dele(id)
	} else {
		return nil
	}
}

type result struct {
	last    int
	success int
	failure int
}

func (f *forwarder) run(sender *mail.Dialer, current int, to []string, delete, dryRun bool) (res result, err error) {
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

		if current > 0 && current >= n {
			continue
		}

		if dryRun {
			success++
			current = n
		} else {
			if forwardErr := f.forward(sender, msg.ID, to, delete); forwardErr != nil {
				failure++
				log.Print(forwardErr)
			} else {
				success++
				if !delete {
					current = n
				}
			}
		}
	}
	res = result{current, success, failure}

	return
}
