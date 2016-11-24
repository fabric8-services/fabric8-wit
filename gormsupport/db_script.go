package gormsupport

import (
	"github.com/jinzhu/gorm"
)

// A DBScript holds a list of functions to be executed against a database
type DBScript struct {
	script []func(db *gorm.DB) error
}

// Run executes all functions in the script, collecting any errors
func (s *DBScript) Run(db *gorm.DB) []error {
	var errs []error
	for _, f := range s.script {
		err := f(db)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Append appends a function to the script
func (s *DBScript) Append(f func(db *gorm.DB) error) {
	s.script = append(s.script, f)
}
