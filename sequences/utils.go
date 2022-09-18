package sequences

import (
	"fmt"
	"github.com/coffeehc/commons/utils"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

var maxFlagMask = int64(-1 ^ (-1 << TimestampLeftShift))

// 从sequence中解析出时间
func ParseSequenceToTime(sequenceID int64) time.Time {
	unixTime := Epoch + (sequenceID >> TimestampLeftShift)
	return time.Unix(0, unixTime*millisecond)
}

func BuildMinSequence(unixNano int64) int64 {
	t1 := unixNano / millisecond
	return (t1 - Epoch) << TimestampLeftShift
}

func BuildMixSequence(unixNano int64) int64 {
	t1 := unixNano / millisecond
	return (t1-Epoch)<<TimestampLeftShift | maxFlagMask
}

func SequenceIdToNo(sequence int64) string {
	t := ParseSequenceToTime(sequence)
	nonce := ^sequence & nonceMask
	tStr := []byte(t.In(utils.TimeLocatioin).Format(utils.Format_TIME_YYYYMMDDhhmmssSSS))
	for i := 15; i < 18; i++ {
		tStr[i-1] = tStr[i]
	}
	return strings.ToUpper(fmt.Sprintf("%s%s", tStr[:17], strconv.FormatInt(nonce, 36)))
}

func SequenceNoToId(sequence string) int64 {
	tStr := make([]byte, 0, 18)
	tStr = append([]byte(sequence[:14]), '.')
	tStr = append(tStr, sequence[14:17]...)
	t, err := time.ParseInLocation(utils.Format_TIME_YYYYMMDDhhmmssSSS, string(tStr), utils.TimeLocatioin)
	if err != nil {
		log.Error("解析编号失败", zap.Error(err))
		return 0
	}
	t1 := t.UnixNano()/millisecond - Epoch
	nonce, err := strconv.ParseUint(strings.ToLower(sequence[17:]), 36, 64)
	if err != nil {
		log.Error("解析编号随机码失败", zap.Error(err))
		return 0
	}
	nonce = ^nonce & nonceMask
	return t1<<TimestampLeftShift | int64(nonce)
}
