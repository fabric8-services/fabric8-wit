package controller_test

import (
	"path/filepath"
	"testing"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
)

func TestListRootOK(t *testing.T) {

	// given
	svc := goa.New("rootService")
	ctrl := controller.NewRootController(svc)

	// when
	_, listRoot := test.ListRootOK(t, svc.Context, svc, ctrl)

	// then
	compareWithGoldenAgnostic(t, filepath.Join("test-files", "root", "list", "ok_root.res.payload.golden.json"), listRoot)
	relationships := listRoot.Data.Relationships
	require.NotNil(t, relationships)
	user := relationships["current_user"]
	require.NotNil(t, user)
}
