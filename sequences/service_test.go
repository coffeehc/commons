package sequences

import (
	"context"
	"testing"
	"time"

	"github.com/coffeehc/base/log"
	"github.com/coffeehc/boot/configuration"
	"github.com/coffeehc/boot/plugin"
	"github.com/coffeehc/boot/testutils"
	"go.uber.org/zap"
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
	EnablePlugin(context.TODO())
	plugin.StartPlugins(s.ctx)
}

func (impl *suite) TestGetDir(c *check.C) {
	testSequence := service.NextID()
	log.Debug("生成出序列", zap.Int64("id", testSequence))
	no := SequenceIdToNo(testSequence)
	// log.Debug("计算出编号",zap.String("no",no))
	id := SequenceNoToId(no)
	log.Debug("计算出编号", zap.String("no", no))
	log.Debug("解析出编号", zap.Int64("id", id))
}
