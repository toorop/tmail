package scope

import (
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/logger"
	"github.com/jinzhu/gorm"
)

var (
	Cfg *config.Config
	DB  gorm.DB
	Log *logger.Logger
)

// TODO check validity de chaque élément
func Init(cfg *config.Config, db gorm.DB, log *logger.Logger) {
	Cfg = cfg
	DB = db
	Log = log
	return
}
