package client

import (
	goaclient "github.com/goadesign/goa/client"
	"net/http"
)

// Client is the alm service client.
type Client struct {
	*goaclient.Client
	SignerJWT goaclient.Signer
}

// New instantiates the client.
func New(c *http.Client) *Client {
	return &Client{
		Client:    goaclient.New(c),
		SignerJWT: &goaclient.JWTSigner{},
	}
}
