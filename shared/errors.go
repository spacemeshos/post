package shared

import (
	"errors"
)

var (
	ErrInitNotStarted   = errors.New("not started")
	ErrInitCompleted    = errors.New("already completed")
	ErrInitNotCompleted = errors.New("not completed")
	ErrProofNotExist    = errors.New("proof doesn't exist")
)
