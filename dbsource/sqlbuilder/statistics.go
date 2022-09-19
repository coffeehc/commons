package sqlbuilder

import (
	"fmt"
	"github.com/coffeehc/commons/utils"
	"time"

	"github.com/coffeehc/base/errors"
)

func (impl *StatisticsTimeRange) BuildTimeStamp(timestampColName string) (string, error) {
	switch impl.Type {
	case RangeType_Minute:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(%s,'%%Y-%%m-%%d %%H:%%i:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName), nil
	case RangeType_MinuteN:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(concat(date(%s), ' ', HOUR (%s), ':', floor(MINUTE(%s)/%d)*%d), '%%Y-%%m-%%d %%H:%%i:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName, timestampColName, timestampColName, impl.Interval, impl.Interval), nil
	case RangeType_Hour:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(%s,'%%Y-%%m-%%d %%H:00:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName), nil
	case RangeType_Day:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(%s,'%%Y-%%m-%%d 00:00:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName), nil
	case RangeType_Week:
		return fmt.Sprintf("DATE_ADD('%s',INTERVAL - WEEKDAY('%s') DAY)", timestampColName, timestampColName), nil
	case RangeType_Month:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(%s,'%%Y-%%m-01 00:00:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName), nil
	case RangeType_Year:
		return fmt.Sprintf("STR_TO_DATE(DATE_FORMAT(%s,'%%Y-01-01 00:00:00'),'%%Y-%%m-%%d %%H:%%i:%%s')", timestampColName), nil
	default:
		return "", errors.MessageError("没有指定时间区间类型")
	}
}

func (impl *StatisticsTimeRange) GetTimestamps() []int64 {
	times := []int64{
		impl.Start,
	}
	for i := impl.Start; i < impl.End; {
		t := time.Unix(0, i)
		switch impl.Type {
		case RangeType_ALL:
			return times
		case RangeType_Minute:
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, utils.TimeLocatioin)
		case RangeType_MinuteN:
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+int(impl.Interval), 0, 0, utils.TimeLocatioin)
		case RangeType_Hour:
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, utils.TimeLocatioin)
		case RangeType_Day:
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, utils.TimeLocatioin)
		case RangeType_Week:
			t = t.AddDate(0, 0, -1*int(t.Weekday()))
			t = time.Date(t.Year(), t.Month(), t.Day()+7, 0, 0, 0, 0, utils.TimeLocatioin)
		case RangeType_Month:
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, utils.TimeLocatioin)
		case RangeType_Year:
			t = time.Date(t.Year()+1, t.Month(), 1, 0, 0, 0, 0, utils.TimeLocatioin)
		default:
			return times
		}
		// log.Debug("时间点",zap.Time("point",t))
		i = t.UnixNano()
		times = append(times, i)
	}
	return times
}
