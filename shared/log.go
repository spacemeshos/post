package shared

type Logger interface {
	Info(format string, args ...any)
	Debug(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
	Panic(format string, args ...any)
}

type DisabledLogger struct{}

func (DisabledLogger) Info(string, ...any)    {}
func (DisabledLogger) Debug(string, ...any)   {}
func (DisabledLogger) Warning(string, ...any) {}
func (DisabledLogger) Error(string, ...any)   {}
func (DisabledLogger) Panic(string, ...any)   {}
