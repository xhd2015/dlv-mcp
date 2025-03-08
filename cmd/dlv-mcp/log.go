package main

import (
	"fmt"
	"io"
	"time"

	"github.com/xhd2015/dlv-mcp/log"
)

type logger struct {
	writer io.Writer
}

var _ log.Logger = &logger{}

func (l *logger) Infof(format string, args ...interface{}) {
	l.writeLog("INFO", fmt.Sprintf(format, args...))
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.writeLog("DEBUG", fmt.Sprintf(format, args...))
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.writeLog("WARN", fmt.Sprintf(format, args...))
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.writeLog("ERROR", fmt.Sprintf(format, args...))
}

func (l *logger) Info(args ...interface{}) {
	l.writeLog("INFO", fmt.Sprint(args...))
}

func (l *logger) Debug(args ...interface{}) {
	l.writeLog("DEBUG", fmt.Sprint(args...))
}

func (l *logger) Warn(args ...interface{}) {
	l.writeLog("WARN", fmt.Sprint(args...))
}

func (l *logger) Error(args ...interface{}) {
	l.writeLog("ERROR", fmt.Sprint(args...))
}

func (l *logger) writeLog(level string, msg string) {
	time := time.Now().Format("2006-01-02 15:04:05")
	l.writer.Write([]byte(time))
	l.writer.Write([]byte(" "))
	l.writer.Write([]byte(level))
	l.writer.Write([]byte(" "))
	l.writer.Write([]byte(msg))
	l.writer.Write([]byte("\n"))
}
