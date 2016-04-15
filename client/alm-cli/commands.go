package main

import (
	"github.com/almighty/almighty-core/client"
	"github.com/goadesign/goa"
	goaclient "github.com/goadesign/goa/client"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"log"
	"os"
)

type (
	// AuthorizeLoginCommand is the command line data structure for the authorize action of login
	AuthorizeLoginCommand struct {
	}
	// ShowVersionCommand is the command line data structure for the show action of version
	ShowVersionCommand struct {
	}
)

// Run makes the HTTP request corresponding to the AuthorizeLoginCommand command.
func (cmd *AuthorizeLoginCommand) Run(c *client.Client, args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		path = "/api/login/authorize"
	}
	logger := goa.NewStdLogger(log.New(os.Stderr, "", log.LstdFlags))
	ctx := goa.WithLogger(context.Background(), logger)
	resp, err := c.AuthorizeLogin(ctx, path)
	if err != nil {
		goa.LogError(ctx, "failed", "err", err)
		return err
	}

	goaclient.HandleResponse(c.Client, resp, PrettyPrint)
	return nil
}

// RegisterFlags registers the command flags with the command line.
func (cmd *AuthorizeLoginCommand) RegisterFlags(cc *cobra.Command, c *client.Client) {
}

// Run makes the HTTP request corresponding to the ShowVersionCommand command.
func (cmd *ShowVersionCommand) Run(c *client.Client, args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		path = "/api/version"
	}
	logger := goa.NewStdLogger(log.New(os.Stderr, "", log.LstdFlags))
	ctx := goa.WithLogger(context.Background(), logger)
	resp, err := c.ShowVersion(ctx, path)
	if err != nil {
		goa.LogError(ctx, "failed", "err", err)
		return err
	}

	goaclient.HandleResponse(c.Client, resp, PrettyPrint)
	return nil
}

// RegisterFlags registers the command flags with the command line.
func (cmd *ShowVersionCommand) RegisterFlags(cc *cobra.Command, c *client.Client) {
	c.SignerJWT.RegisterFlags(cc)
}
