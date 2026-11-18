package logger

type ModuleLogger struct {
	hub    *Hub
	module string
}

func (m *ModuleLogger) Log(lvl LoggerLevel, msg string) {
	m.hub.Log(lvl, "["+m.module+"] :: "+msg)
}

func (m *ModuleLogger) Logf(lvl LoggerLevel, lazy func() string) {
	m.hub.Logf(lvl, func() string { return "[" + m.module + "] :: " + lazy() })
}

func (m *ModuleLogger) Debug(msg string)          { m.Log(DEBUG, msg) }
func (m *ModuleLogger) Debugf(lazy func() string) { m.Logf(DEBUG, lazy) }
func (m *ModuleLogger) Info(msg string)           { m.Log(INFO, msg) }
func (m *ModuleLogger) Warning(msg string)        { m.Log(WARNING, msg) }
func (m *ModuleLogger) Error(msg string)          { m.Log(ERROR, msg) }
func (m *ModuleLogger) Critical(msg string)       { m.Log(CRITICAL, msg) }
