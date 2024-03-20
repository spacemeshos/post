package postrs

// #cgo LDFLAGS: -lpost
// #include <stdlib.h>
// #include "post.h"
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
	logMux sync.RWMutex
	log    *zap.Logger

	oncer sync.Once

	levelMap = map[zapcore.Level]C.Level{
		zapcore.DebugLevel:  C.Level_Trace,
		zapcore.InfoLevel:   C.Level_Info,
		zapcore.WarnLevel:   C.Level_Warn,
		zapcore.ErrorLevel:  C.Level_Error,
		zapcore.DPanicLevel: C.Level_Error,
		zapcore.PanicLevel:  C.Level_Error,
		zapcore.FatalLevel:  C.Level_Error,
	}

	zapLevelMap = map[C.Level]zapcore.Level{
		C.Level_Error: zapcore.ErrorLevel,
		C.Level_Warn:  zapcore.WarnLevel,
		C.Level_Info:  zapcore.InfoLevel,
		C.Level_Debug: zapcore.DebugLevel,
		C.Level_Trace: zapcore.DebugLevel,
	}
)

func setLogCallback(logger *zap.Logger) {
	level, ok := levelMap[logger.Level()]
	if !ok {
		logger.Error("failed to map zap log level to C log level", zap.Stringer("level", logger.Level()))
		return
	}

	logMux.Lock()
	log = logger
	logMux.Unlock()

	oncer.Do(func() {
		C.set_logging_callback(level, C.callback(C.logCallback))
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

	logMux.RLock()
	defer logMux.RUnlock()
	log.Log(zapLevelMap[record.level], msg, fields...)
}
