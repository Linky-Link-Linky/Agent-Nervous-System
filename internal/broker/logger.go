package broker

// DiscardLogger discards all log messages.
type DiscardLogger struct{}

func (DiscardLogger) Infof(format string, args ...interface{})  {}
func (DiscardLogger) Errorf(format string, args ...interface{}) {}
