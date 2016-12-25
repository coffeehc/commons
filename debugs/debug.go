package debugs

import (
	"github.com/coffeehc/logger"
	"runtime/debug"
)

//DebugPanic print Panic message
func DebugPanic(printStick bool) {
	if err := recover(); err != nil {
		logger.Error("发生错误:%#v", err)
		if printStick {
			debug.PrintStack()
		}
		panic(err)
	}
}
