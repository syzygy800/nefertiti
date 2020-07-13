package empty

import (
	"strconv"
)

func AsString(v interface{}) string {
	if str, ok := v.(string); ok {
		return str
	}
	if i, ok := v.(int); ok {
		return strconv.Itoa(i)
	}
	if i64, ok := v.(int64); ok {
		return strconv.FormatInt(i64, 10)
	}
	if u64, ok := v.(uint64); ok {
		return strconv.FormatUint(u64, 10)
	}
	if i32, ok := v.(int32); ok {
		return strconv.FormatInt(int64(i32), 10)
	}
	if u32, ok := v.(uint32); ok {
		return strconv.FormatUint(uint64(u32), 10)
	}
	if f64, ok := v.(float64); ok {
		return strconv.FormatFloat(f64, 'f', -1, 64)
	}
	return ""
}

func AsFloat64(v interface{}) float64 {
	if f64, ok := v.(float64); ok {
		return f64
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	if i32, ok := v.(int32); ok {
		return float64(i32)
	}
	if i64, ok := v.(int64); ok {
		return float64(i64)
	}
	if str, ok := v.(string); ok {
		if out, err := strconv.ParseFloat(str, 64); err == nil {
			return out
		}
	}
	return 0
}
