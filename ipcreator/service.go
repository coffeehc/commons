package ipcreator

import (
	"embed"
	"github.com/coffeehc/commons/coder"
	"github.com/coffeehc/commons/cryptos"
	"github.com/coffeehc/commons/utils"
)

//go:embed ips.json
var ipJson embed.FS

var ips map[string][]*ipRange = make(map[string][]*ipRange)
var allIpRanges []*ipRange = make([]*ipRange, 0)

func init() {
	data, err := ipJson.ReadFile("ips.json")
	if err != nil {
		panic(err)
	}
	err = coder.JsonCoder.Unmarshal(data, &ips)
	if err != nil {
		panic(err)
	}
	allIpRanges = make([]*ipRange, 0)
	for _, vs := range ips {
		for _, v := range vs {
			v.Max, err = utils.IpToInt64(v.MaxStr)
			if err != nil {
				panic(err)
			}
			v.Min, err = utils.IpToInt64(v.MinStr)
			if err != nil {
				panic(err)
			}
		}
		allIpRanges = append(allIpRanges, vs...)
		//log.Debug("读取ip池", zap.String("province", k), zap.Int("count", len(vs)))
	}
}

func GetProvinceRandomIp(provinceName string) string {
	list := ips[provinceName]
	ipRange := list[cryptos.GetRandInt(len(list)-1, 0)]
	ip := cryptos.GetRandInt64()%(ipRange.Max-ipRange.Min) + ipRange.Min
	ipStr, _ := utils.Int64ToIp(ip)
	return ipStr
}

func GetRandomIp() string {
	ipRange := allIpRanges[cryptos.GetRandInt(len(allIpRanges)-1, 0)]
	ip := cryptos.GetRandInt64()%(ipRange.Max-ipRange.Min) + ipRange.Min
	ipStr, _ := utils.Int64ToIp(ip)
	return ipStr
}
