package forwarder

import (
	"encoding/base64"

	"github.com/Azure/go-ntlmssp"
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
