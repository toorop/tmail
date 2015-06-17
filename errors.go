package main

import (
	"errors"
)

var (
	ErrNonAsciiCharDetected = errors.New("email must contains only 7-bit ASCII characters")
)
