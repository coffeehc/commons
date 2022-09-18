package sequences

import "time"

const (
	//纪元时间(毫秒) //自定义起始时间
	Epoch    = int64(1444281954363)
	dcIDBits = 3
	MaxDCID  = -1 ^ (-1 << dcIDBits)
	dcMask   = -1 ^ (-1 << dcIDBits)
	//节点Id位数
	nodeIDBits = 5
	//最大节点Id
	MaxNodeID = -1 ^ (-1 << nodeIDBits)
	nodeMask  = -1 ^ (-1 << nodeIDBits)
	nodeShift = dcIDBits
	//序列号位数
	indexBits = 10
	//序列号偏移量
	indexShift = nodeIDBits + dcIDBits
	indexMask  = -1 ^ (-1 << indexBits)
	//时间戳偏移量
	TimestampLeftShift = indexBits + nodeIDBits + dcIDBits
	nonceMask          = -1 ^ (-1 << TimestampLeftShift)
	millisecond        = int64(time.Millisecond)
)

// SequenceService 接口定义
type SequenceService interface {
	//元年时间,使用默认配置epoch(毫秒级别)
	GetEpoch() int64
	//获取节点Id
	GetNodeID() int64
	//获取下一个Id
	NextID() int64
	//解析sequence
	ParseSequence(sequence int64) *Sequence
	//给出指定时间戳内最小的 SequenceId, 时间戳单位毫秒
	MinID(timestemp int64) int64
}
