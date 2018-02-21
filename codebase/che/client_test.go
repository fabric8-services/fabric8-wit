package che_test

import (
	"github.com/fabric8-services/fabric8-wit/codebase/che"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCodebaseToMap(t *testing.T) {
	// Test for 'ide url'
	hrefIde := "https://che.prod-preview.openshift.io/user/wksp-3a2"
	methodIde := "GET"
	relIde := "ide url"

	// Test for 'self link'
	hrefSelf := "https://che.prod-preview.openshift.io/wsmaster/api/workspace/workspaceij5jeym0zs9ve2q12"
	methodSelf := "GET"
	relSelf := "self link"

	workspaceIdeLink := che.WorkspaceLink{hrefIde, methodIde, relIde}
	workspaceSelfLink := che.WorkspaceLink{hrefSelf, methodSelf, relSelf}

	workspaceResponse := che.WorkspaceResponse{"id", "description", che.WorkspaceConfig{"workspaceName"}, "RUNNING", []che.WorkspaceLink{workspaceIdeLink, workspaceSelfLink}}

	ideLink := workspaceResponse.GetHrefByRelOfWorkspaceLink(che.IdeUrlRel)
	selfLink := workspaceResponse.GetHrefByRelOfWorkspaceLink(che.SelfLinkRel)

	assert.Equal(t, hrefIde, ideLink)
	assert.Equal(t, hrefSelf, selfLink)
}
