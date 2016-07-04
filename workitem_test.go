package main

import (
	"strconv"
	"testing"
	"flag"
	"fmt"
	"os"

	"github.com/almighty/almighty-core/app"
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
	controller := WorkitemController{db: db}
	_, wi := test.ShowWorkitemOK(t, nil, nil, &controller, strconv.FormatUint(uint64(id), 10))

	if wi == nil {
		t.Errorf("Work Item '%d' not present", id)
	}

	if wi.ID != strconv.FormatUint(uint64(id), 10) {
		t.Errorf("Id should be %d, but is %s", id, wi.ID)
	}
}

func TestCreateWI(t *testing.T) {
	controller := WorkitemController{db: db}
	name:= "some name"
	typeid:="1"
	payload:= app.CreateWorkitemPayload {
		Name: &name,
		TypeID: &typeid,
		Fields: map[string]interface{}{
			"system.owner": "tmaeder",
			"system.state": "open",
		},
	}
	
	_, created := test.CreateWorkitemOK(t, nil, nil, &controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}
