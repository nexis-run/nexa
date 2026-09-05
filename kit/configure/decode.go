package configure

import (
	"fmt"
	"math"
	"reflect"
)

// validateIntegerConversion 拒绝数值到整数字段的截断、溢出和符号丢失
func validateIntegerConversion(from, to reflect.Value) (any, error) {
	kind := to.Kind()
	if kind < reflect.Int || kind > reflect.Uint64 {
		return from.Interface(), nil
	}

	signed := kind <= reflect.Int64
	var invalid bool

	switch from.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number := from.Int()

		if signed {
			invalid = to.OverflowInt(number)
		} else {
			invalid = number < 0 || to.OverflowUint(uint64(number))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number := from.Uint()

		if signed {
			invalid = number > math.MaxInt64 || to.OverflowInt(int64(number))
		} else {
			invalid = to.OverflowUint(number)
		}
	case reflect.Float32, reflect.Float64:
		number := from.Float()
		upper := math.Ldexp(1, to.Type().Bits())
		lower := 0.0

		if signed {
			upper /= 2
			lower = -upper
		}

		invalid = math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number || number < lower || number >= upper
	}

	if invalid {
		return nil, fmt.Errorf("%v 不能无损转换为 %s", from.Interface(), to.Type())
	}

	return from.Interface(), nil
}
