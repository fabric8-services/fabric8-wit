package controller

import (
	"github.com/fabric8-services/fabric8-wit/codebase/che"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWorkspaceLinks(t *testing.T) {
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

	workspaceResponse := &che.WorkspaceResponse{
		ID:          "id",
		Description: "description",
		Config: che.WorkspaceConfig{
			Name: "workspaceName",
		},
		Status: "RUNNING",
		Links:  []che.WorkspaceLink{workspaceIdeLink, workspaceSelfLink}}

	ideLink := workspaceResponse.GetHrefByRelOfWorkspaceLink(che.IdeUrlRel)
	selfLink := workspaceResponse.GetHrefByRelOfWorkspaceLink(che.SelfLinkRel)

	assert.Equal(t, hrefIde, ideLink)
	assert.Equal(t, hrefSelf, selfLink)
}

func TestWorkspaceProjects(t *testing.T) {
	codebaseURL := "https://github.com/xtermjs/xterm.js"
	// First WorkspaceProject attributes
	locationOfFirstProject := "https://github.com/xtermjs/xterm.js"
	branchOfFirstProject := "v3"

	// Second WorkspaceProject attributes
	locattionOfSecondProject := "https://github.com/apache/incubator-ripple"
	branchOfSecondProject := "master"

	firstProject := che.WorkspaceProject{
		Source: che.ProjectSource{
			Location: locationOfFirstProject,
			Parameters: che.ProjectSourceParameters{
				Branch: branchOfFirstProject,
			},
		},
	}

	secondProject := che.WorkspaceProject{
		Source: che.ProjectSource{
			Location: locattionOfSecondProject,
			Parameters: che.ProjectSourceParameters{
				Branch: branchOfSecondProject,
			},
		},
	}

	workspaceResponse := &che.WorkspaceResponse{
		ID:          "id",
		Description: "description",
		Config: che.WorkspaceConfig{
			Name:     "workspaceName",
			Projects: []che.WorkspaceProject{firstProject, secondProject},
		},
		Status: "RUNNING"}

	codebaseBranch := getBranch(workspaceResponse.Config.Projects, codebaseURL)
	// Codebase branch should be equal to the first project branch
	require.Equal(t, branchOfFirstProject, codebaseBranch)
}
