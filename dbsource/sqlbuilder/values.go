package sqlbuilder

import "time"

func NewValueWithStringArray(array []string) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType:    ValueType_StringArray,
		StringValues: array,
	}
}

func NewValueWithIntArray(array []int64) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueType_IntArray,
		IntValues: array,
	}
}

func NewValueWithDoubleArray(array []float64) *Value {
	if len(array) == 0 {
		return nil
	}
	return &Value{
		ValueType:    ValueType_DoubleArray,
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
		ValueType: ValueType_TimeArray,
		IntValues: times,
	}
}

func NewValueWithString(v string, skipEmpty bool) *Value {
	if skipEmpty && v == "" {
		return nil
	}
	return &Value{
		ValueType:   ValueType_String,
		StringValue: v,
	}
}

func NewValueWithInt(v int64, skipEmpty bool) *Value {
	if skipEmpty && v == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueType_Int,
		IntValue:  v,
	}
}

func NewValueWithDouble(v float64, skipEmpty bool) *Value {
	if skipEmpty && v == 0 {
		return nil
	}
	return &Value{
		ValueType:   ValueType_Doubel,
		DoubleValue: v,
	}
}

func NewValueWithTime(v time.Time, skipEmpty bool) *Value {
	if skipEmpty && v.UnixNano() == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueType_Time,
		IntValue:  v.UnixNano(),
	}
}

func NewValueWithBool(v bool) *Value {
	return &Value{
		ValueType: ValueType_Bool,
		BoolValue: v,
	}
}

func NewValueWithStatus(v int64, skipEmpty bool) *Value {
	if skipEmpty && v <= 0 {
		return nil
	}
	return &Value{
		ValueType: ValueType_Status,
		IntValue:  v,
	}
}

func NewValueWithStatuses(v []int64) *Value {
	if len(v) == 0 {
		return nil
	}
	return &Value{
		ValueType: ValueType_Statuses,
		IntValues: v,
	}
}
