package controller_test

import (
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

type brokenFileSystemSimulator struct{}

func (s brokenFileSystemSimulator) Asset(fileName string) ([]byte, error) {
	return nil, errs.Errorf("failed to file name %s", fileName)
}

func TestListRootOK(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		// given
		svc := goa.New("rootService")
		ctrl := controller.NewRootController(svc)
		// when
		res, listRoot := test.ListRootOK(t, svc.Context, svc, ctrl)
		// then
		compareWithGoldenAgnostic(t, filepath.Join("test-files", "root", "list", "ok_root.res.payload.golden.json"), listRoot)
		compareWithGoldenAgnostic(t, filepath.Join("test-files", "root", "list", "ok.res.headers.golden.json"), res.Header())
		relationships := listRoot.Data.Relationships
		require.NotNil(t, relationships)
		user := relationships["current_user"]
		require.NotNil(t, user)
	})
	t.Run("file not found", func(t *testing.T) {
		// given
		svc := goa.New("rootService")
		ctrl := controller.NewRootController(svc)
		ctrl.FileHandler = brokenFileSystemSimulator{}
		// when
		res, jerrs := test.ListRootNotFound(t, svc.Context, svc, ctrl)
		// then
		compareWithGoldenAgnostic(t, filepath.Join("test-files", "root", "list", "not_found.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join("test-files", "root", "list", "not_found.res.headers.golden.json"), res.Header())
	})
}
