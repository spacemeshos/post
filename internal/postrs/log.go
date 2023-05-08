package postrs

import "C"

import (
	"github.com/spacemeshos/post/shared"
)

var log shared.Logger

//export logCallback
type logCallback func(level C.int, message *C.char)

// callBackLogger returns a logCallback that logs to the provided logger. The logCallback is
// exported to C and can be used as a callback function by Rust.
//
// TODO(mafa): proof of work, needs completion.
func callBackLogger(log shared.Logger) logCallback {
	return func(level C.int, message *C.char) {
		switch level {
		case 0:
			log.Debug(C.GoString(message))
		case 1:
			log.Info(C.GoString(message))
		case 2:
			log.Warning(C.GoString(message))
		case 3:
			log.Error(C.GoString(message))
		default:
			log.Error(C.GoString(message))
		}
	}
}
