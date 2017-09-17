package codebase_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestCodebaseToMap(t *testing.T) {
	branch := "task-101"
	repo := "golang-project"
	file := "main.go"
	line := 200
	cb := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}

	codebaseMap := cb.ToMap()
	require.NotNil(t, codebaseMap)
	assert.Equal(t, repo, codebaseMap[codebase.RepositoryKey])
	assert.Equal(t, branch, codebaseMap[codebase.BranchKey])
	assert.Equal(t, file, codebaseMap[codebase.FileNameKey])
	assert.Equal(t, line, codebaseMap[codebase.LineNumberKey])
}

func TestNewCodebase(t *testing.T) {
	// Test for empty map
	codebaseMap := map[string]interface{}{}
	cb, err := codebase.NewCodebaseContent(codebaseMap)
	require.NotNil(t, err)
	assert.Equal(t, "", cb.Repository)
	assert.Equal(t, "", cb.Branch)
	assert.Equal(t, "", cb.FileName)
	assert.Equal(t, 0, cb.LineNumber)

	// test for all values in codebase
	branch := "task-101"
	repo := "https://github.com/pranavgore09/go-tutorial.git"
	file := "main.go"
	line := 200
	codebaseMap = map[string]interface{}{
		codebase.RepositoryKey: repo,
		codebase.BranchKey:     branch,
		codebase.FileNameKey:   file,
		codebase.LineNumberKey: line,
	}
	cb, err = codebase.NewCodebaseContent(codebaseMap)
	require.Nil(t, err)
	assert.Equal(t, repo, cb.Repository)
	assert.Equal(t, branch, cb.Branch)
	assert.Equal(t, file, cb.FileName)
	assert.Equal(t, line, cb.LineNumber)
}

func TestIsValid(t *testing.T) {
	cb := codebase.Content{
		Repository: "https://github.com/pranavgore09/go-tutorial.git",
	}
	assert.Nil(t, cb.IsValid())

	cb = codebase.Content{}
	assert.NotNil(t, cb.IsValid())
}

func TestInvalidRepo(t *testing.T) {
	cb := codebase.Content{
		Repository: "https://other-than-github.com/pranavgore09/go-tutorial",
	}
	assert.NotNil(t, cb.IsValid())
}

func TestRepoValidURL(t *testing.T) {
	// following list is taken from
	// https://github.com/jonschlinkert/is-git-url/blob/master/test.js
	validURLs := []string{
		"git://github.com/ember-cli/ember-cli.git#ff786f9f",
		"git://github.com/ember-cli/ember-cli.git#gh-pages",
		"git://github.com/ember-cli/ember-cli.git#master",
		"git://github.com/ember-cli/ember-cli.git#Quick-Fix",
		"git://github.com/ember-cli/ember-cli.git#quick_fix",
		"git://github.com/ember-cli/ember-cli.git#v0.1.0",
		"git://host.xz/path/to/repo.git/",
		"git://host.xz/~user/path/to/repo.git/",
		"git@192.168.101.127:user/project.git",
		"git@github.com:user/project.git",
		"git@github.com:user/some-project.git",
		"git@github.com:user/some-project.git",
		"git@github.com:user/some_project.git",
		"git@github.com:user/some_project.git",
		"http://192.168.101.127/user/project.git",
		"http://github.com/user/project.git",
		"http://host.xz/path/to/repo.git/",
		"https://192.168.101.127/user/project.git",
		"https://github.com/user/project.git",
		"https://host.xz/path/to/repo.git/",
		"https://username::;*%$:@github.com/username/repository.git",
		"https://username:$fooABC@:@github.com/username/repository.git",
		"https://username:password@github.com/username/repository.git",
		"ssh://host.xz/path/to/repo.git/",
		"ssh://host.xz/path/to/repo.git/",
		"ssh://host.xz/~/path/to/repo.git",
		"ssh://host.xz/~user/path/to/repo.git/",
		"ssh://host.xz:port/path/to/repo.git/",
		"ssh://user@host.xz/path/to/repo.git/",
		"ssh://user@host.xz/path/to/repo.git/",
		"ssh://user@host.xz/~/path/to/repo.git",
		"ssh://user@host.xz/~user/path/to/repo.git/",
		"ssh://user@host.xz:port/path/to/repo.git/",
	}

	for _, url := range validURLs {
		cb := codebase.Content{
			Repository: url,
		}
		assert.True(t, cb.IsRepoValidURL(), "valid URL %s detected as invalid", url)
	}

	invalidURLs := []string{
		"",
		"/path/to/repo.git/",
		"file:///path/to/repo.git/",
		"file://~/path/to/repo.git/",
		"git@github.com:user/some_project.git/foo",
		"git@github.com:user/some_project.gitfoo",
		"host.xz:/path/to/repo.git/",
		"host.xz:path/to/repo.git",
		"host.xz:~user/path/to/repo.git/",
		"path/to/repo.git/",
		"rsync://host.xz/path/to/repo.git/",
		"user@host.xz:/path/to/repo.git/",
		"user@host.xz:path/to/repo.git",
		"user@host.xz:~user/path/to/repo.git/",
		"~/path/to/repo.git",
	}
	for _, url := range invalidURLs {
		cb := codebase.Content{
			Repository: url,
		}
		assert.False(t, cb.IsRepoValidURL(), "invalid URL %s not detected as valid", url)
	}
}

type TestCodebaseRepository struct {
	gormtestsupport.DBTestSuite
	clean func()
}

func TestRunCodebaseRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestCodebaseRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestCodebaseRepository) SetupTest() {
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestCodebaseRepository) TearDownTest() {
	test.clean()
}

func (test *TestCodebaseRepository) TestListCodebases() {
	// given
	fxt := tf.NewTestFixture(test.T(), test.DB,
		tf.Codebases(2, func(fxt *tf.TestFixture, idx int) error {
			fxt.Codebases[idx].URL = "git@github.com:fabric8-services/fabric8-wit.git"
			if idx == 1 {
				fxt.Codebases[idx].URL = "git@github.com:aslakknutsen/fabric8-wit.git"
			}
			return nil
		}),
	)
	// when
	offset := 0
	limit := 1
	codebases, _, err := codebase.NewCodebaseRepository(test.DB).List(context.Background(), fxt.Codebases[0].SpaceID, &offset, &limit)
	// then
	require.Nil(test.T(), err)
	require.Len(test.T(), codebases, 1)
	require.Equal(test.T(), fxt.Codebases[0].URL, codebases[0].URL)
}

func (test *TestCodebaseRepository) TestExistsCodebase() {
	repo := codebase.NewCodebaseRepository(test.DB)
	test.T().Run("codebase exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, test.DB, tf.Codebases(1))
		// when
		err := repo.CheckExists(context.Background(), fxt.Codebases[0].ID.String())
		// then
		require.Nil(t, err)
	})

	test.T().Run("codebase doesn't exist", func(t *testing.T) {
		// when
		err := repo.CheckExists(context.Background(), uuid.NewV4().String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (test *TestCodebaseRepository) TestLoadCodebase() {
	// given
	fxt := tf.NewTestFixture(test.T(), test.DB, tf.Codebases(1))
	repo := codebase.NewCodebaseRepository(test.DB)
	// when
	loadedCodebase, err := repo.Load(context.Background(), fxt.Codebases[0].ID)
	require.Nil(test.T(), err)
	assert.Equal(test.T(), fxt.Codebases[0].ID, loadedCodebase.ID)
	require.NotNil(test.T(), fxt.Codebases[0].StackID)
	assert.Equal(test.T(), *fxt.Codebases[0].StackID, *loadedCodebase.StackID)
	assert.Equal(test.T(), fxt.Codebases[0].LastUsedWorkspace, loadedCodebase.LastUsedWorkspace)
}
