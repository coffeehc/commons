package sqlbuilder

import "time"

func NewValueWithStringArray(array []string) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType:    ValueTypeStringArray,
		StringValues: array,
	}
}

func NewValueWithIntArray(array []int64) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueTypeIntArray,
		IntValues: array,
	}
}

func NewValueWithDoubleArray(array []float64) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType:    ValueTypeDoubleArray,
		DoubleValues: array,
	}
}

func NewValueWithTimeArray(array []time.Time) *Value {
	if len(array) == 0 {
		return nil
	}
	times := make([]int64, 0, len(array))
	for _, t := range array {
		times = append(times, t.UnixNano())
	}
	return &Value{
		ValueType: ValueTypeTimeArray,
		IntValues: times,
	}
}

func NewValueWithString(v string, skipEmpty bool) *Value {
	if skipEmpty && v == "" {
		return nil
	}
	return &Value{
		ValueType:   ValueTypeString,
		StringValue: v,
	}
}

func NewValueWithInt(v int64, skipEmpty bool) *Value {
	if skipEmpty && v == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueTypeInt,
		IntValue:  v,
	}
}

func NewValueWithDouble(v float64, skipEmpty bool) *Value {
	if skipEmpty && v == 0 {
		return nil
	}
	return &Value{
		ValueType:   ValueTypeDoubel,
		DoubleValue: v,
	}
}

func NewValueWithTime(v time.Time, skipEmpty bool) *Value {
	if skipEmpty && v.UnixNano() == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueTypeTime,
		IntValue:  v.UnixNano(),
	}
}

func NewValueWithBool(v bool) *Value {
	return &Value{
		ValueType: ValueTypeBool,
		BoolValue: v,
	}
}

func NewValueWithStatus(v int64, skipEmpty bool) *Value {
	if skipEmpty && v <= 0 {
		return nil
	}
	return &Value{
		ValueType: ValueTypeStatus,
		IntValue:  v,
	}
}

func NewValueWithStatuses(v []int64) *Value {
	if len(v) == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueTypeStatuses,
		IntValues: v,
	}
}
