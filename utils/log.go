package utils

/*
A wrapper of standard library logger, support log level
*/

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	LogErrorLevel int = 0
	LogWarnLevel  int = 1
	LogInfoLevel  int = 2
	LogDebugLevel int = 3

	fatalLevelStr   = "TAG[Fatal] "
	fatalLevelStrln = "TAG[Fatal] %v\n"
	errLevelStr     = "TAG[Error] "
	errLevelStrln   = "TAG[Error] %v\n"
	warnLevelStr    = "TAG[Warn] "
	warnLevelStrln  = "TAG[Warn] %v\n"
	infoLevelStr    = "TAG[Info] "
	infoLevelStrln  = "TAG[Info] %v\n"
	debugLevelStr   = "TAG[Debug] "
	debugLevelStrln = "TAG[Debug] %v\n"

	defaultCallDepth = 2
)

var (
	stdoutLog *Logger
	stdout    *log.Logger
	logLevel  = LogDebugLevel
)

func init() {
	stdout = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	stdoutLog = NewLogger("")
}

func SetLogLevel(level int) {
	logLevel = level
}

func GetLogLevel() int {
	return logLevel
}

func GetStdoutLog() *Logger {
	return stdoutLog
}

type Logger struct {
	*log.Logger
	fatalLevelStr   string
	fatalLevelStrln string
	errLevelStr     string
	errLevelStrln   string
	warnLevelStr    string
	warnLevelStrln  string
	infoLevelStr    string
	infoLevelStrln  string
	debugLevelStr   string
	debugLevelStrln string
}

func NewLogger(tag string) *Logger {
	prefix := tag
	if len(tag) != 0 {
		prefix = "[" + tag + "]"
	}
	result := &Logger{
		Logger:          stdout,
		fatalLevelStr:   strings.Replace(fatalLevelStr, "TAG", prefix, -1),
		fatalLevelStrln: strings.Replace(fatalLevelStrln, "TAG", prefix, -1),
		errLevelStr:     strings.Replace(errLevelStr, "TAG", prefix, -1),
		errLevelStrln:   strings.Replace(errLevelStrln, "TAG", prefix, -1),
		warnLevelStr:    strings.Replace(warnLevelStr, "TAG", prefix, -1),
		warnLevelStrln:  strings.Replace(warnLevelStrln, "TAG", prefix, -1),
		infoLevelStr:    strings.Replace(infoLevelStr, "TAG", prefix, -1),
		infoLevelStrln:  strings.Replace(infoLevelStrln, "TAG", prefix, -1),
		debugLevelStr:   strings.Replace(debugLevelStr, "TAG", prefix, -1),
		debugLevelStrln: strings.Replace(debugLevelStrln, "TAG", prefix, -1),
	}

	return result
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.fatalLevelStr+format, v...))
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.fatalLevelStrln, v...))
	os.Exit(1)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.errLevelStr+format, v...))
}

func (l *Logger) Errorln(v ...interface{}) {
	l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.errLevelStrln, v...))
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if LogWarnLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.warnLevelStr+format, v...))
	}
}

func (l *Logger) Warnln(v ...interface{}) {
	if LogWarnLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.warnLevelStrln, v...))
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if LogInfoLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.infoLevelStr+format, v...))
	}
}

func (l *Logger) Infoln(v ...interface{}) {
	if LogInfoLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.infoLevelStrln, v...))
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if LogDebugLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.debugLevelStr+format, v...))
	}
}

func (l *Logger) Debugln(v ...interface{}) {
	if LogDebugLevel <= logLevel {
		l.Logger.Output(defaultCallDepth, fmt.Sprintf(l.debugLevelStrln, v...))
	}
}
