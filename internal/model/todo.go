package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Todo represents a single to-do item stored in the database.
type Todo struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Task      string    `gorm:"type:text;not null" json:"task"`
	DueDate   time.Time `gorm:"not null" json:"due_date"`
	Completed bool      `gorm:"default:false" json:"completed"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName tells GORM which table to use for this model.
func (Todo) TableName() string {
	return "todos"
}

// BeforeCreate hook ensures each record gets a UUID if one has not been set.
func (t *Todo) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
