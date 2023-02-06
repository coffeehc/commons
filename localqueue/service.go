package localqueue

import (
	"github.com/coffeehc/base/log"
	"github.com/nsqio/go-diskqueue"
	"os"
	"time"
)

func NewQueue(name, queueDir string) diskqueue.Interface {
	os.MkdirAll(queueDir, os.ModePerm)
	return diskqueue.New(
		name, // 很重要，关系到队列缓存文件的名字
		queueDir,
		1024*1024*512*1, //  单个文件512M
		4,               // 最小4字节
		1024*1024*4,     // 最大消息4M
		100,             // 每100次写入同步
		5*time.Second,   // 每5s同步

		func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
			if lvl == diskqueue.INFO {
				log.GetLogger().Sugar().Infof(f, args)
			} else if lvl == diskqueue.WARN {
				log.GetLogger().Sugar().Warnf(f, args)
			} else if lvl > diskqueue.WARN {
				log.GetLogger().Sugar().Errorf(f, args)
			}
		})
}
