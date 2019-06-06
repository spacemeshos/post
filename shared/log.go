package shared

type Logger interface {
	Info(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
	Panic(format string, args ...interface{})
}

type DisabledLogger struct{}

func (DisabledLogger) Info(string, ...interface{})    {}
func (DisabledLogger) Debug(string, ...interface{})   {}
func (DisabledLogger) Warning(string, ...interface{}) {}
func (DisabledLogger) Error(string, ...interface{})   {}
func (DisabledLogger) Panic(string, ...interface{})   {}
