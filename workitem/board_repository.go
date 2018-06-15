package workitem

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/spacetemplate"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// BoardRepository encapsulates storage & retrieval of work item boards.
type BoardRepository interface {
	repository.Exister
	Create(ctx context.Context, board Board) (*Board, error)
	Load(ctx context.Context, groupID uuid.UUID) (*Board, error)
	List(ctx context.Context, spaceTemplateID uuid.UUID) ([]*Board, error)
}

// NewBoardRepository creates a wi type group repository based on gorm.
func NewBoardRepository(db *gorm.DB) *GormBoardRepository {
	return &GormBoardRepository{db}
}

// GormBoardRepository implements BoardRepository using gorm.
type GormBoardRepository struct {
	db *gorm.DB
}

// Load returns the board for the given id.
func (r *GormBoardRepository) Load(ctx context.Context, boardID uuid.UUID) (*Board, error) {
	log.Debug(ctx, map[string]interface{}{"board_id": boardID}, "loading work item board ")
	res := Board{}
	db := r.db.Model(&res).Where("id=?", boardID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"board_id": boardID}, "work item board not found")
		return nil, errors.NewNotFoundError("work item board", boardID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	columns, err := r.loadColumns(ctx, res.ID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	res.Columns = columns
	return &res, nil
}

// loadColumns loads all columns associated with the given board.
func (r *GormBoardRepository) loadColumns(ctx context.Context, boardID uuid.UUID) ([]BoardColumn, error) {
	columns := []BoardColumn{}
	db := r.db.Model(&columns).Where("board_id=?", boardID).Order("column_order ASC").Find(&columns)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"board_id": boardID}, "work item board columns not found")
		return nil, errors.NewNotFoundError("work item board columns of board", boardID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	return columns, nil
}

// List returns all boards for the given space template ID
// ordered by their position value.
func (r *GormBoardRepository) List(ctx context.Context, spaceTemplateID uuid.UUID) ([]*Board, error) {
	log.Debug(ctx, map[string]interface{}{"space_template_id": spaceTemplateID}, "loading work item boards for space template")
	// check space template exists
	if err := spacetemplate.NewRepository(r.db).CheckExists(ctx, spaceTemplateID); err != nil {
		return nil, errors.NewNotFoundError("space template", spaceTemplateID.String())
	}
	res := []*Board{}
	db := r.db.Model(&res).Where("space_template_id=?", spaceTemplateID).Find(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"space_template_id": spaceTemplateID}, "work item boards not found")
		return nil, errors.NewNotFoundError("work item boards for space template", spaceTemplateID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	for _, board := range res {
		columns, err := r.loadColumns(ctx, board.ID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		board.Columns = columns
	}
	return res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormBoardRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	log.Debug(ctx, map[string]interface{}{"board_id": id}, "checking if work item board exists")
	return repository.CheckExists(ctx, r.db, Board{}.TableName(), id)
}

// Create creates a new work item board in the repository
func (r *GormBoardRepository) Create(ctx context.Context, b Board) (*Board, error) {
	if len(b.Columns) <= 0 {
		return nil, errors.NewBadParameterError("columns", b.Columns).Expected("not empty")
	}
	if b.ID == uuid.Nil {
		b.ID = uuid.NewV4()
	}
	db := r.db.Create(&b)
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	log.Debug(ctx, map[string]interface{}{"board_id": b.ID}, "created work item board")
	// Create entries for each column in the column list
	for _, column := range b.Columns {
		if column.ID == uuid.Nil {
			column.ID = uuid.NewV4()
		}
		column.BoardID = b.ID
		db = db.Create(&column)
		if db.Error != nil {
			return nil, errors.NewInternalError(ctx, db.Error)
		}
	}
	return &b, nil
}
