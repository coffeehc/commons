package ipcreator_test

import (
	"context"
	"github.com/coffeehc/commons/ipcreator"
	"testing"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/configuration"
	"github.com/coffeehc/boot/plugin"
	"github.com/coffeehc/boot/testutils"
	"gopkg.in/check.v1"
)

var _ = check.Suite(&suite{})

func Test(t *testing.T) { check.TestingT(t) }

type suite struct {
	dir       string // 测试用的临时目录
	f         string // 测试用的临时文件
	ctx       context.Context
	cancelFun context.CancelFunc
}

func (s *suite) TearDownSuite(c *check.C) {
	defer log.GetLogger().Sync()
}

func (s *suite) SetUpSuite(c *check.C) {
	testutils.InitTestConfig()
	s.ctx, s.cancelFun = context.WithTimeout(context.TODO(), time.Second*30)
	configuration.InitConfiguration(context.TODO(), configuration.ServiceInfo{
		ServiceName: "test",
	})
	ipcreator.EnablePlugin(s.ctx)
	plugin.StartPlugins(s.ctx)
}

func (impl *suite) TestGetDir(c *check.C) {
	service := ipcreator.GetService()
	ip := service.GetProvinceRandomIp("上海")
	log.Debug(ip)
}
