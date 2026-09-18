package number

import (
	"cmp"
	"fmt"
	"strconv"
)

func Equal(a, b any) bool {
	ret, _ := EqualE(a, b)
	return ret
}

func LessThan(a, b any) bool {
	ret, _ := LessThanE(a, b)
	return ret
}

func GreaterThan(a, b any) bool {
	ret, _ := GreaterThanE(a, b)
	return ret
}

func LessOrEqual(a, b any) bool {
	ret, _ := LessOrEqualE(a, b)
	return ret
}

func GreaterOrEqual(a, b any) bool {
	ret, _ := GreaterOrEqualE(a, b)
	return ret
}

func EqualE(a, b any) (bool, error) {
	ret, err := compare(a, b)
	return ret == 0, err
}

func LessThanE(a, b any) (bool, error) {
	ret, err := compare(a, b)
	return ret == -1, err
}

func GreaterThanE(a, b any) (bool, error) {
	ret, err := compare(a, b)
	return ret == 1, err
}

func LessOrEqualE(a, b any) (bool, error) {
	ret, err := compare(a, b)
	return ret == -1 || ret == 0, err
}

func GreaterOrEqualE(a, b any) (bool, error) {
	ret, err := compare(a, b)
	return ret == 1 || ret == 0, err
}

func compare(a, b any) (int, error) {
	af, err := toFloat64(a)
	if err != nil {
		return 0, err
	}

	bf, err := toFloat64(b)
	if err != nil {
		return 0, err
	}

	return cmp.Compare(af, bf), nil
}

func toFloat64(v any) (float64, error) {
	switch vv := v.(type) {
	case int:
		return float64(vv), nil
	case int8:
		return float64(vv), nil
	case int16:
		return float64(vv), nil
	case int32:
		return float64(vv), nil
	case int64:
		return float64(vv), nil
	case uint:
		return float64(vv), nil
	case uint8:
		return float64(vv), nil
	case uint16:
		return float64(vv), nil
	case uint32:
		return float64(vv), nil
	case uint64:
		return float64(vv), nil
	case float32:
		return float64(vv), nil
	case float64:
		return vv, nil
	case string:
		ff, err := strconv.ParseFloat(vv, 64)
		if err != nil {
			return 0, err
		}
		return ff, nil
	default:
		return 0, fmt.Errorf("unsupported type: %v", v)
	}
}
