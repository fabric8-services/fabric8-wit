package controller

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/configuration"

	"github.com/dnaeon/go-vcr/recorder"
	goauuid "github.com/goadesign/goa/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeleteOpenShiftResource(t *testing.T) {

	t.Run("ok", func(t *testing.T) {
		t.Run("with delete pipeline success", func(t *testing.T) {
			// given
			r, err := recorder.New("../test/data/deployments/deployments_delete_space.ok")
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

			// when
			err = deleteOpenShiftResource(client, config, ctx, spaceID)
			// then
			require.NoError(t, err)
		})

		t.Run("with delete pipeline failure", func(t *testing.T) {
			// given
			r, err := recorder.New("../test/data/deployments/deployments_delete_space.ok.no_pipeline")
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

			// when
			err = deleteOpenShiftResource(client, config, ctx, spaceID)
			// then
			require.NoError(t, err)
		})
	})

	t.Run("failure", func(t *testing.T) {
		t.Run("space not found", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in f", r)
				}
			}()
			// given
			r, err := recorder.New("../test/data/deployments/deployments_delete_space.404.space_not_found")
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

			// when
			err = deleteOpenShiftResource(client, config, ctx, spaceID)
			// then
			require.Error(t, err)
		})
	})
}

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
