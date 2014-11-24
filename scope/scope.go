package scope

import (
	"github.com/Toorop/tmail/config"
	"github.com/jinzhu/gorm"
)

type Scope struct {
	Cfg *config.Config
	DB  gorm.DB
}

// New return pointer to a scope struct
// Helper
func New(cfg *config.Config, db gorm.DB) *Scope {
	return &Scope{cfg, db}
}
