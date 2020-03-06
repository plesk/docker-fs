package log

import (
	"fmt"
	golog "log"
	"strings"
)

type LogLevel int

const (
	Critical LogLevel = iota
	Error
	Warning
	Info
	Debug
	Trace
)

func (l LogLevel) String() string {
	switch l {
	case Critical:
		return "critical"
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Info:
		return "info"
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	}
	panic(fmt.Errorf("Unknown LogLevel: %d", l))
}

var (
	// Default level
	Level     = Warning
	allLevels = []LogLevel{Critical, Error, Warning, Info, Debug, Trace}
)

func SetLevel(level string) error {
	lvl := strings.ToLower(level)
	for _, l := range allLevels {
		if strings.HasPrefix(l.String(), lvl) {
			Level = l
			return nil
		}
	}
	return fmt.Errorf("Unknown LogLevel: %v", level)
}

func Printf(format string, v ...interface{}) {
	for _, lvl := range allLevels {
		if strings.HasPrefix(format, "["+lvl.String()+"] ") {
			if Level >= lvl {
				golog.Printf(format, v...)
			}
			return
		}
	}
	// Warning level by default
	if Level >= Warning {
		golog.Printf(format, v...)
	}
}

//pass through
func Fatal(v ...interface{}) {
	golog.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	golog.Fatalf(format, v...)
}
