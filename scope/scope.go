package scope

import (
	"github.com/jinzhu/gorm"
	"log"
)

type Scope struct {
	TRACE *log.Logger
	INFO  *log.Logger
	ERROR *log.Logger
	db    gorm.DB
}

func New(t, i, r *log.Logger, db gorm.DB) (*Scope, error) {
	return &Scope{t, i, r, db}, nil
}
