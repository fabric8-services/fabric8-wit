package repository

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/errors"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

type Exister interface {
	// Exists returns true as the first parameter if the object with the given ID exists;
	// otherwise false is returned. The error is either nil in case of success or not nil
	// if there has been an issue.
	Exists(ctx context.Context, id string) (bool, error)
}

// Exists returns true if an item exists in the database table with a given ID
func Exists(ctx context.Context, db *gorm.DB, tableName string, id string) (bool, error) {
	var exists bool
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				id=$1
				AND deleted_at IS NULL
		)`, tableName)

	err := db.CommonDB().QueryRow(query, id).Scan(&exists)
	if err == nil && !exists {
		return exists, goa.ErrNotFound(errors.NewNotFoundError(tableName, id).Error())
	}
	if err != nil {
		return false, errors.NewInternalError(ctx, errs.Wrapf(err, "unable to verify if %s exists", tableName))
	}
	return exists, nil
}
