package bard

import (
	"log"
	"os"
)

/**
	比较晚出现的文件,主要用于
	控制debug时的调试输出问题
	还有之后的异常或一些特殊错误的记录问题-----记录日志文件
 */

const (
	LOG_INFO = "INFO\t"
	LOG_WARNING = "WARNING\t"
	LOG_EXCEPTION = "EXCEPTION\t"
)
var Deb = NewDebug()
var Slog = NewSysLog()
func Logf(format string, arg ...interface{}) {
	Deb.Printf(format, arg...)
	Slog.Printf(format, arg...)
}

func Logln(arg ...interface{}) {
	Deb.Println(arg...)
	Slog.Println(arg...)
}

func Logff(format string, level string, arg ...interface{}) {
	Deb.Printf(format, arg...)
	Slog.SetPrefix(level)
	Slog.Printf(format, arg)
	// 最后还原
	Slog.SetPrefix(LOG_INFO)
}

type Logger interface {
	Printf(format string, args ...interface{})
	Println(arg ...interface{})
	SetPrefix(pre string)
}

type Debug struct {
	Open bool
	Debug Logger
}

func NewDebug() *Debug {
	d := &Debug{false, nil}
	d.Debug = log.New(os.Stderr, "Debug\t", log.Ltime)
	return d
}

func (d *Debug) SetPrefix(pre string) {
	d.Debug.SetPrefix(pre)
}

func (d *Debug) Printf(format string, args ...interface{}) {
	if d.Open {
		d.Debug.Printf(format, args...)
	}
}

func (d *Debug) Println(args ...interface{}) {
	if d.Open {
		d.Debug.Println(args...)
	}
}

// 系统日志记录
// 日志一般分级 信息 警告 异常
func NewSysLog() *Debug {
	// TODO 文件的位子应该可以动态更改
	logfile, e := os.OpenFile("./bard.log", os.O_APPEND|os.O_CREATE, os.ModeAppend)
	if e != nil {
		// 交由外层函数处理吧
		panic(e)
		return nil
	}
	logger := log.New(logfile, LOG_INFO, log.LstdFlags)
	var slog = &Debug{true, logger}
	return slog
}
