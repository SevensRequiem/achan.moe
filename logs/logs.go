package logs

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	DebugLogger *log.Logger
	ErrorLogger *log.Logger
	FatalLogger *log.Logger
)

func init() {
	logFile := &lumberjack.Logger{
		Filename:   "app.log",
		MaxSize:    10,   // megabytes
		MaxBackups: 3,    // retain 3 backups
		MaxAge:     28,   // days
		Compress:   true, // compress the backups
	}

	InfoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLogger = log.New(logFile, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(logFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	FatalLogger = log.New(logFile, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Info(format string, v ...interface{}) {
	InfoLogger.Output(2, fmt.Sprintf(format, v...))
}

func Warn(format string, v ...interface{}) {
	WarnLogger.Output(2, fmt.Sprintf(format, v...))
}

func Debug(format string, v ...interface{}) {
	DebugLogger.Output(2, fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	ErrorLogger.Output(2, fmt.Sprintf(format, v...))
}

func Fatal(format string, v ...interface{}) {
	FatalLogger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
