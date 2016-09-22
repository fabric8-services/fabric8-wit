package remoteworkitem

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestLookupProvider(t *testing.T) {
	ts1 := trackerSchedule{TrackerType: ProviderGithub}
	tp1 := LookupProvider(ts1)
	if tp1 == nil {
		t.Error("nil provider")
	}
	ts2 := trackerSchedule{TrackerType: ProviderJira}
	tp2 := LookupProvider(ts2)
	if tp2 == nil {
		t.Error("nil provider")
	}
	ts3 := trackerSchedule{TrackerType: "unknown"}
	tp3 := LookupProvider(ts3)
	if tp3 != nil {
		t.Error("non-nil provider")
	}
}

func TestNewScheduler(t *testing.T) {
	db := new(gorm.DB)
	s := NewScheduler(db)
	if s.db != db {
		t.Error("DB not set as an attribute")
	}
	s.Stop()
}
