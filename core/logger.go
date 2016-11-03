package core

// NSQLogger is a logger for Nsq
type NSQLogger struct{}

// NewNSQLogger return a new NSQLogger
func NewNSQLogger() *NSQLogger {
	return new(NSQLogger)
}

// Output implements nsq.logger.Output interface
func (n *NSQLogger) Output(calldepth int, s string) error {
	Logger.Debug(s)
	return nil
}
