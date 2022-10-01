package ipcreator

import (
	"context"
	"embed"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/coder"
	"github.com/coffeehc/commons/cryptos"
	"github.com/coffeehc/commons/utils"
	"go.uber.org/zap"
)

//go:embed ips.json
var ipJson embed.FS

type Service interface {
	GetRandomIp() string
	GetProvinceRandomIp(provinceName string) string
}

func newService(ctx context.Context) Service {
	impl := &serviceImpl{
		ips: make(map[string][]*ipRange),
	}
	return impl
}

type serviceImpl struct {
	ips         map[string][]*ipRange
	allIpRanges []*ipRange
}

func (impl *serviceImpl) GetProvinceRandomIp(provinceName string) string {
	list := impl.ips[provinceName]
	ipRange := list[cryptos.GetRandInt(len(list)-1, 0)]
	ip := cryptos.GetRandInt64()%(ipRange.Max-ipRange.Min) + ipRange.Min
	ipStr, _ := utils.Int64ToIp(ip)
	return ipStr
}

func (impl *serviceImpl) GetRandomIp() string {
	ipRange := impl.allIpRanges[cryptos.GetRandInt(len(impl.allIpRanges)-1, 0)]
	ip := cryptos.GetRandInt64()%(ipRange.Max-ipRange.Min) + ipRange.Min
	ipStr, _ := utils.Int64ToIp(ip)
	return ipStr
}

func (impl *serviceImpl) Start(ctx context.Context) error {
	defer func() {
		if e := recover(); e != nil {
			log.Error("错误", zap.Any("error", e))
		}
	}()
	data, err := ipJson.ReadFile("ips.json")
	if err != nil {
		return err
	}
	err = coder.JsonCoder.Unmarshal(data, &impl.ips)
	if err != nil {
		return err
	}
	impl.allIpRanges = make([]*ipRange, 0)
	for _, vs := range impl.ips {
		for _, v := range vs {
			v.Max, err = utils.IpToInt64(v.MaxStr)
			if err != nil {
				return err
			}
			v.Min, err = utils.IpToInt64(v.MinStr)
			if err != nil {
				return err
			}
		}
		impl.allIpRanges = append(impl.allIpRanges, vs...)
		//log.Debug("读取ip池", zap.String("province", k), zap.Int("count", len(vs)))
	}
	return nil
}

func (impl *serviceImpl) Stop(ctx context.Context) error {
	return nil
}
