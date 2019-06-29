package bard

import (
	"testing"
)

func TestNewSysLog(t *testing.T) {
	//defer func() {
	//	r := recover()
	//	log.Println(r)
	//}()

	slog := NewSysLog()
	slog.Println("bard log test!")
}
