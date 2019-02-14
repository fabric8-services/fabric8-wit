package controller

import (
	"context"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/configuration"

	"github.com/dnaeon/go-vcr/recorder"
	goauuid "github.com/goadesign/goa/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeleteCodebases(t *testing.T) {
	r, err := recorder.New("../test/data/codebases/codebases_delete_space.ok")
	require.NoError(t, err)
	defer r.Stop()

	spaceID, err := goauuid.FromString("aec5f659-0680-4633-8599-5f14f1deeabc")
	require.NoError(t, err)
	ctx := context.Background()
	config, err := configuration.New("")
	require.NoError(t, err)

	client := &http.Client{
		Transport: r.Transport,
	}

	err = deleteCodebases(client, config, ctx, spaceID)
	require.NoError(t, err)
}
