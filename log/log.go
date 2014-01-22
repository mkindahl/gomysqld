// Support for logging using different log levels.
//
// This package use the log package but use functions that log message
// based on a priority set.
package log

import "log"

// Priority is a type to enumerate the logging levels. Higher priority
// levels, such as "error" have lower numbers, while lower priorities,
// such as "info" have higher numbers.
type Priority int

const (
	PRIORITY_ERROR = iota
	PRIORITY_WARNING
	PRIORITY_INFO
	PRIORITY_DEBUG
)

var priority Priority = PRIORITY_WARNING

// SetLevel set the log level priority to pri. Any messages for that
// priority or higher will then be printed, so priority "warning" will
// print both "warning" and "error", but not "info".
func SetPriority(pri Priority) {
	priority = pri
}

func Debug(a ...interface{}) {
	if priority >= PRIORITY_DEBUG {
		log.Print(a...)
	}
}

func Debugln(a ...interface{}) {
	if priority >= PRIORITY_DEBUG {
		log.Println(a...)
	}
}

func Debugf(format string, a ...interface{}) {
	if priority >= PRIORITY_DEBUG {
		log.Printf(format, a...)
	}
}

func Info(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Print(a...)
	}
}

func Infoln(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Println(a...)
	}
}

func Infof(format string, a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Printf(format, a...)
	}
}

func Warning(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Print(a...)
	}
}

func Warningln(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Println(a...)
	}
}

func Warningf(format string, a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Printf(format, a...)
	}
}

func Error(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Print(a...)
	}
}

func Errorln(a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Println(a...)
	}
}

func Errorf(format string, a ...interface{}) {
	if priority >= PRIORITY_INFO {
		log.Printf(format, a...)
	}
}
