package main

import (
	"strconv"
	"testing"
	"flag"
	"fmt"
	"os"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"

	"github.com/jinzhu/gorm"
)

var db *gorm.DB
var id uint

func TestMain(m *testing.M) {
	var dbhost *string= flag.String("dbhost", "", "-dbhost <hostname>")
	flag.Parse()
	if "" == *dbhost {
		flag.Usage()
		os.Exit(-1)
	}
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", *dbhost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()

	item := models.WorkItem{
		Fields:  models.Fields{"foo": "bar"},
		Name:    "foobar",
		Type:    "73",
		Version: 1}
	db.Table("work_items").Create(&item)
	id= item.ID
	// Migrate the schema
	migration.Perform(db)
	m.Run()
	db.Table("work_items").Delete(&item, "")
}

func TestGetWork(t *testing.T) {
	if db == nil {
		panic("no db")
	}
	controller := WorkitemController{db: db}
	if controller.db == nil {
		panic("no db 2")
	}
	resp := test.ShowWorkitemOK(t, &controller, strconv.FormatUint(uint64(id), 10))

	if resp == nil {
		t.Errorf("Work Item '%d' not present", id)
	}

	if resp.ID != strconv.FormatUint(uint64(id), 10) {
		t.Errorf("Id should be %d, but is %s", id, resp.ID)
	}
}
