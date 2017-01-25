package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Identity ddenDescribes a unique Person with the ALM
type Data struct {
	gormsupport.Lifecycle
	ID   uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Path string
	Data workitem.Fields `sql:"type:jsonb"`
}

// TableName implements gorm.tabler
func (w Data) TableName() string {
	return "userspace_data"
}

// UserspaceController implements the userspace resource.
type UserspaceController struct {
	*goa.Controller
	db *gorm.DB
}

// NewUserspaceController creates a userspace controller.
func NewUserspaceController(service *goa.Service, db *gorm.DB) *UserspaceController {
	db.AutoMigrate(&Data{})

	return &UserspaceController{Controller: service.NewController("UserspaceController"), db: db}
}

// Create runs the create action.
func (c *UserspaceController) Create(ctx *app.CreateUserspaceContext) error {
	return models.Transactional(c.db, func(db *gorm.DB) error {
		data := Data{
			ID:   uuid.NewV4(),
			Path: ctx.RequestURI,
			Data: workitem.Fields(ctx.Payload),
		}

		err := c.db.Create(&data).Error
		if err != nil {
			goa.LogError(ctx, "error adding Identity", "error", err.Error())
			return ctx.InternalServerError()
		}

		return ctx.Created()
	})
}
