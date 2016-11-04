package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var DB *gorm.DB
var rwiScheduler *remoteworkitem.Scheduler

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {

		DB, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer DB.Close()

		// Make sure the database is populated with the correct types (e.g. system.bug etc.)
		if configuration.GetPopulateCommonTypes() {
			if err := models.Transactional(DB, func(tx *gorm.DB) error {
				return migration.PopulateCommonTypes(context.Background(), tx, models.NewWorkItemTypeRepository(tx))
			}); err != nil {
				panic(err.Error())
			}

		}

		// RemoteWorkItemScheduler now available for all other test cases
		rwiScheduler = remoteworkitem.NewScheduler(DB)
	}
	os.Exit(m.Run())
}

func TestNewWorkitemController(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	assert.Panics(t, func() {
		NewWorkitemController(goa.New("Test service"), nil)
	})
}

func TestParseInts(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	integers, err := parseInts(nil)
	assert.Equal(t, nil, err)
	assert.Equal(t, []int{}, integers)

	str := "1, 2, foo"
	_, err = parseInts(&str)
	assert.NotNil(t, err)
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test parsing error in parseInts
	str := "1000, foo"
	integers, length, err := parseLimit(&str)
	assert.NotNil(t, err)
	assert.Equal(t, 0, length)
	assert.Nil(t, integers)

	// Test length = 1
	str = "1000"
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 1000, length)
	assert.Nil(t, integers)

	// Test empty string
	str = ""
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 100, length)
	assert.Nil(t, integers)
}

func TestSetPagingLinks(t *testing.T) {
	links := &app.PagingLinks{}
	setPagingLinks(links, "", 0, 0, 1, 0)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0,page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "prefix", 0, 0, 1, 0)
	assert.Equal(t, "prefix?page[offset]=0,page[limit]=1", *links.First)
	assert.Equal(t, "prefix?page[offset]=0,page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 0, 1, 1)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 0)
	assert.Equal(t, "?page[offset]=0,page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0,page[limit]=0", *links.Last)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 1)
	assert.Equal(t, "?page[offset]=0,page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 2, 1, 1)
	assert.Equal(t, "?page[offset]=0,page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0,page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 3, 4, 4)
	assert.Equal(t, "?page[offset]=0,page[limit]=3", *links.First)
	assert.Equal(t, "?page[offset]=3,page[limit]=4", *links.Last)
	assert.Equal(t, "?page[offset]=3,page[limit]=4", *links.Next)
	assert.Equal(t, "?page[offset]=0,page[limit]=3", *links.Prev)
}
