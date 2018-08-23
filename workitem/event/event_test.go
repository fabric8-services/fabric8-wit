package event

import (
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
)

func TestList_FilterByRevisionID(t *testing.T) {
	type args struct {
		revisionID uuid.UUID
	}
	u1 := uuid.NewV4()
	u2 := uuid.NewV4()
	u3 := uuid.NewV4()
	tests := []struct {
		name string
		l    List
		args args
		want List
	}{
		{"empty list",
			List{},
			args{uuid.NewV4()},
			List{},
		},
		{"revision found twice",
			List{Event{RevisionID: u1, Name: "first"}, Event{RevisionID: u2, Name: "second"}, Event{RevisionID: u3, Name: "thrird"}, Event{RevisionID: u3, Name: "fourth"}, Event{RevisionID: u2, Name: "fifth"}},
			args{u2},
			List{Event{RevisionID: u2, Name: "second"}, Event{RevisionID: u2, Name: "fifth"}},
		},
		{"no events belongs to revision",
			List{Event{RevisionID: u1, Name: "first"}, Event{RevisionID: u2, Name: "second"}, Event{RevisionID: u2, Name: "thrird"}},
			args{u3},
			List{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.FilterByRevisionID(tt.args.revisionID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List.FilterByRevisionID() = %v, want %v", got, tt.want)
			}
		})
	}
}
