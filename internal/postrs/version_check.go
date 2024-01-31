package postrs

// #cgo LDFLAGS: -lpost
// #include "post.h"
import "C"

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	// regexp matching supported versions of post-rs library
	SUPPORTED_VERSION = `0\.7\.(\d+)` // 0.7.*
	// Set this env variable to "true" or "1" to disable version check.
	DISABLE_CKECK_ENV = "LIBPOST_DISABLE_VERSION_CHECK"
)

func init() {
	checkDisabledEnv := strings.ToLower(os.Getenv(DISABLE_CKECK_ENV))
	if checkDisabledEnv == "true" || checkDisabledEnv == "1" {
		return
	}
	version := Version()
	if !regexp.MustCompile(SUPPORTED_VERSION).Match([]byte(version)) {
		msgFmt := `Unsupported version of post library (post.dll/libpost.dll/libpost.dylib):
	got: "%s"
	supported: "%s"`
		panic(fmt.Sprintf(msgFmt, version, SUPPORTED_VERSION))
	}
}

func Version() string {
	return C.GoString(C.version())
}
