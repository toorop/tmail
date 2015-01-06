package scope

import (
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/logger"
	"github.com/jinzhu/gorm"
)

const (
	Time822 = "02 Jan 2006 15:04:05 -0700" // "02 Jan 06 15:04 -0700"
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
