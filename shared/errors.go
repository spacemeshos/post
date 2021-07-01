package shared

import (
	"errors"
	"fmt"
)

var (
	ErrInitNotStarted   = errors.New("not started")
	ErrInitCompleted    = errors.New("already completed")
	ErrInitNotCompleted = errors.New("not completed")
	ErrProofNotExist    = errors.New("proof doesn't exist")
)

type ConfigMismatchError struct {
	Param    string
	Expected string
	Found    string
	DataDir  string
}

func (err ConfigMismatchError) Error() string {
	return fmt.Sprintf("`%v` config mismatch; expected: %v, found: %v, datadir: %v",
		err.Param, err.Expected, err.Found, err.DataDir)
}
