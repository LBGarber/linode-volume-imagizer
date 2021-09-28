package builder

import (
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
	"net/http"
)

type Imagizer struct {
	client *linodego.Client
}

func NewImagizer(token string) *Imagizer {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)

	return &Imagizer{client: &client}
}
