package logger

type ModuleLogger struct {
	hub    *Hub
	module string
}

func (m *ModuleLogger) Log(lvl LoggerLevel, msg string) {
	m.hub.Log(lvl, "["+msg+"]")
}

func (m *ModuleLogger) Debug(msg string)    { m.Log(DEBUG, msg) }
func (m *ModuleLogger) Info(msg string)     { m.Log(INFO, msg) }
func (m *ModuleLogger) Warning(msg string)  { m.Log(WARNING, msg) }
func (m *ModuleLogger) Error(msg string)    { m.Log(ERROR, msg) }
func (m *ModuleLogger) Critical(msg string) { m.Log(CRITICAL, msg) }
