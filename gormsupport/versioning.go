package gormsupport

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/jinzhu/gorm"
)

// Versioning can be embedded into model structs that want to have a Version
// column which will automatically be incremented before each UPDATE, set to 0
// on CREATE, and checked for compatibility on each UPDATE.
//
// For the first creation of a model the initial version will always be
// overwritten with 0 nomatter what the user specified in the model itself. The
// model itself is not changed in any cases, just the DB query for INSERT and
// UPDATE is touched.
//
// We also add
//
//    AND version=<VERSION-OF-THE-MODEL>
//
// to the WHERE conditions of the UPDATE part.
type Versioning struct {
	Version int `json:"version"`
}

// BeforeUpdate is a GORM callback (see http://doc.gorm.io/callbacks.html) that
// will be called before updating the model. We use it to automatically
// increment the version number before saving the model and to check for version
// compatibility by adding this condition to the WHERE clause of the UPDATE:
//
//    AND version=<VERSION-OF-THE-MODEL>
func (v *Versioning) BeforeUpdate(scope *gorm.Scope) error {
	scope.Search.Where(fmt.Sprintf(`"%s"."version"=?`, scope.TableName()), v.Version)
	return scope.SetColumn("version", v.Version+1)
}

// BeforeCreate is a GORM callback (see http://doc.gorm.io/callbacks.html) that
// will be called before creating the model. We use it to automatically
// have the first version of the model set to 0.
func (v *Versioning) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("version", 0)
}

// BeforeDelete is a GORM callback (see http://doc.gorm.io/callbacks.html) that
// will be called before soft-deleting the model. We use it to check for version
// compatibility by adding this condition to the WHERE clause of the deletion:
//
//    AND version=<VERSION-OF-THE-MODEL>
func (v *Versioning) BeforeDelete(scope *gorm.Scope) error {
	scope.Search.Where(fmt.Sprintf(`"%s"."version"=?`, scope.TableName()), v.Version)
	return nil
}

// Ensure Versioning implements the Equaler interface
var _ convert.Equaler = Versioning{}
var _ convert.Equaler = (*Versioning)(nil)

// Equal returns true if two Versioning objects are equal; otherwise false is
// returned.
func (v Versioning) Equal(u convert.Equaler) bool {
	other, ok := u.(Versioning)
	if !ok {
		return false
	}
	return v.Version == other.Version
}
