package sequences

import (
	"sync"
	"time"

	"github.com/coffeehc/base/errors"
)

type sequenceServiceImpl struct {
	// 节点ID
	dcID          int64
	nodeID        int64
	mutex         *sync.Mutex
	index         int64
	lastTimestamp int64
}

func NewSequenceService(dataCenterID, nodeID int64) (SequenceService, error) {
	if dataCenterID > MaxDCID {
		return nil, errors.SystemError("数据中心ID超过最大值7")
	}
	if nodeID > MaxNodeID {
		return nil, errors.SystemError("NodeID超过最大值7")
	}
	return &sequenceServiceImpl{
		dcID:          dataCenterID,
		nodeID:        nodeID,
		mutex:         new(sync.Mutex),
		index:         0,
		lastTimestamp: 0,
	}, nil
}

func (impl *sequenceServiceImpl) GetEpoch() int64 {
	return Epoch
}

func (impl *sequenceServiceImpl) GetNodeID() int64 {
	return impl.nodeID
}

func (impl *sequenceServiceImpl) MinID(timestamp int64) int64 {
	return (timestamp-Epoch)<<TimestampLeftShift | 0<<indexShift
}

func (impl *sequenceServiceImpl) getTimeStampAndIndex() (int64, int64) {
	impl.mutex.Lock()
	index := impl.index
	t := time.Now()
	timestamp := t.UnixNano() / millisecond
	if impl.lastTimestamp == timestamp {
		index = (index + 1) & indexMask
		if index == 0 {
			// 当前毫秒内计数满了，则等待下一秒
			timestamp = tilNextMillis(impl.lastTimestamp * millisecond)
		}
	} else {
		index = 0
	}
	impl.index = index
	impl.lastTimestamp = timestamp
	impl.mutex.Unlock()
	return timestamp, index
}

func (impl *sequenceServiceImpl) NextID() int64 {
	t, i := impl.getTimeStampAndIndex()
	return (t-Epoch)<<TimestampLeftShift | i<<indexShift | impl.nodeID<<nodeShift | impl.dcID
}

func (impl *sequenceServiceImpl) ParseSequence(sequenceID int64) *Sequence {
	unixTime := Epoch + (sequenceID >> TimestampLeftShift)
	index := sequenceID >> indexShift & indexMask
	nodeID := sequenceID >> nodeShift & nodeMask
	dcID := sequenceID & dcMask
	return &Sequence{
		Id:           sequenceID,
		CreateTimeMS: unixTime,
		NodeID:       nodeID,
		Index:        index,
		DcID:         dcID,
	}
}

// 等待下一毫秒
func tilNextMillis(lastTimestamp int64) int64 {
	timestamp := time.Now().UnixNano()
	for timestamp < lastTimestamp {
		time.Sleep(time.Duration(lastTimestamp - timestamp))
		timestamp = time.Now().UnixNano()
	}
	return timestamp / millisecond
}
