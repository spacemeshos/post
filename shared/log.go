package shared

type Logger interface {
	Info(format string, args ...any)
	Debug(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
	Panic(format string, args ...any)
}

type NoopLogger struct{}

func (NoopLogger) Info(string, ...any)    {}
func (NoopLogger) Debug(string, ...any)   {}
func (NoopLogger) Warning(string, ...any) {}
func (NoopLogger) Error(string, ...any)   {}
func (NoopLogger) Panic(string, ...any)   {}
