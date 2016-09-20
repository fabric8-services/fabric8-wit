package models

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestGormTransactionSupport(t *testing.T) {
	resource.Require(t, resource.Database)

	var db *gorm.DB

	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
	if err != nil {
		t.Fatal("Cannot connect to DB", err)
	}
	defer db.Close()

	ts := NewGormTransactionSupport(db)
	assert.Nil(t, ts.tx)
	assert.Equal(t, ts.db, db)
	assert.Nil(t, ts.TX())

	err = ts.Begin()
	assert.Nil(t, err)
	assert.NotNil(t, ts.tx)
	assert.Equal(t, ts.TX(), ts.tx)
	ts.Commit()
	assert.Nil(t, ts.tx)
	assert.Nil(t, ts.TX())

	ts.Begin()
	ts.Rollback()
	assert.Nil(t, ts.tx)
	assert.Nil(t, ts.TX())
}
