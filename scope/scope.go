package scope

import (
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/logger"
	"github.com/jinzhu/gorm"
)

type Scope struct {
	Cfg *config.Config
	DB  gorm.DB
	Log *logger.Logger
}

// New return pointer to a scope struct
// Helper
func New(cfg *config.Config, db gorm.DB, log *logger.Logger) *Scope {
	return &Scope{cfg, db, log}
}
