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

func Info(v ...interface{}) {
	InfoLogger.Output(2, fmt.Sprintln(v...))
}

func Warn(v ...interface{}) {
	WarnLogger.Output(2, fmt.Sprintln(v...))
}

func Debug(v ...interface{}) {
	DebugLogger.Output(2, fmt.Sprintln(v...))
}

func Error(v ...interface{}) {
	ErrorLogger.Output(2, fmt.Sprintln(v...))
}

func Fatal(v ...interface{}) {
	FatalLogger.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}
