package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "prover.h"
//
// // forward declarations for callback C functions
// void logCallback(ExternCRecord* record);
// typedef void (*callback)(const struct ExternCRecord*);
import "C"

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log   *zap.Logger
	oncer sync.Once

	levelMap = map[zapcore.Level]C.Level{
		zapcore.DebugLevel:  C.Debug,
		zapcore.InfoLevel:   C.Info,
		zapcore.WarnLevel:   C.Warn,
		zapcore.ErrorLevel:  C.Error,
		zapcore.DPanicLevel: C.Error,
		zapcore.PanicLevel:  C.Error,
		zapcore.FatalLevel:  C.Error,
	}

	zapLevelMap = map[C.Level]zapcore.Level{
		C.Error: zapcore.ErrorLevel,
		C.Warn:  zapcore.WarnLevel,
		C.Info:  zapcore.InfoLevel,
		C.Debug: zapcore.DebugLevel,
		C.Trace: zapcore.DebugLevel,
	}
)

func setLogCallback(logger *zap.Logger) {
	oncer.Do(func() {
		C.set_logging_callback(levelMap[logger.Level()], C.callback(C.logCallback))
		log = logger
	})
}

//export logCallback
func logCallback(record *C.ExternCRecord) {
	msg := C.GoStringN(record.message.ptr, (C.int)(record.message.len))
	fields := []zap.Field{
		zap.String("module", C.GoStringN(record.module_path.ptr, (C.int)(record.module_path.len))),
		zap.String("file", C.GoStringN(record.file.ptr, (C.int)(record.file.len))),
		zap.Int64("line", int64(record.line)),
	}

	log.Log(zapLevelMap[record.level], msg, fields...)
}
