package utils

import (
	"time"
)

var TimeLocatioin = time.FixedZone("CST", 8*3600)

const Format_TIME_ALL = "2006-01-02 15:04:05.999999999"
const Format_TIME_YYYY_MM_DD_hh_mm_ss = "2006-01-02 15:04:05"
const Format_TIME_YYYY_MM_DD_hh_mm_ss_SSS = "2006-01-02 15:04:05.999"
const Format_TIME_YYYY_MM_DD_hh_mm_ = "2006-01-02 15:04"
const Format_TIME_YYYY_MM_DD = "2006-01-02"
const Format_TIME_YYYYMMDDhhmmss = "20060102150405"
const Format_TIME_YYYYMMDDhhmmssSSS = "20060102150405.000"

var NullStruct = &struct{}{}
var NullArray = make([]*struct{}, 0)

var TimeOneDay = int64(time.Hour * 24)
var TimeZero = time.Unix(0, 0)

func GetTimeTodayZero() int64 {
	now := time.Now().In(TimeLocatioin)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func GetTimeYesterdayRange() (int64, int64) {
	endTime := GetTimeTodayZero()
	return endTime - TimeOneDay, endTime
}

func GetTimeWeekZero() int64 {
	now := time.Now().In(TimeLocatioin)
	now = now.AddDate(0, 0, -1*int(now.Weekday()))
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func GetTimeMonthZero() int64 {
	now := time.Now().In(TimeLocatioin)
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func GetTimeYearZero() int64 {
	now := time.Now().In(TimeLocatioin)
	return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func ToDayZero(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func ToWeekZero(t time.Time) int64 {
	t = t.AddDate(0, 0, -1*int(t.Weekday()))
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func ToMonthZero(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, TimeLocatioin).UnixNano()
}

func ToYearZero(t time.Time) int64 {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, TimeLocatioin).UnixNano()
}
