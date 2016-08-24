package remotetracker

import (
	"github.com/jinzhu/gorm"
  // "github.com/robfig/cron"
)

// Schedule fetch and import of remote tracker items
func Schedule(db *gorm.DB) {
  tq := fetchTrackerQueries(db)
  for _, v := range tq{
    scheduleFetchAndImport(v)
  }
}

func fetchTrackerQueries(db *gorm.DB) []string  {
  return []string{"", "", ""}
}

func scheduleFetchAndImport( s string ) {

}
