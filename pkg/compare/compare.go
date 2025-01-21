package compare

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/spf13/cast"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
)

type (
	Comparator interface {
		Compare(operation Operation, left, right any) (bool, error)
	}

	Operation uint8

	comparator struct{}
)

const (
	// EqualsOperation the left, right will transform to string then compare two string.
	EqualsOperation Operation = iota + 1
	// NotEqualsOperation like above.
	NotEqualsOperation
	// ContainsOperation exists two possibilities:
	// (string, string), will use strings.Contains(left, right)
	// ([], string), will search in slice.
	// every item will transform to string.
	ContainsOperation
	// ContainsLowerCasedOperation transforms both left and right params
	//	// into lower cases before comparing.
	ContainsLowerCasedOperation
	// NotContainsOperation like above.
	NotContainsOperation
	// NotContainsLowerCasedOperation transforms both left and right params
	// into lower cases before comparing.
	NotContainsLowerCasedOperation
	// StringStartWithOperation strings.HasPrefix(left, right)
	StringStartWithOperation
	// StringEndWithOperation strings.HasSuffix(left, right)
	StringEndWithOperation
	// TimeBeforeOperation left and right should transform time.Time, left < right
	TimeBeforeOperation
	// TimeAfterOperation left and right should transform time.Time, left > right
	TimeAfterOperation
	// TimeDayAgoOperation left transform time.Time first, then compare with (right - n*time.Day)
	TimeDayAgoOperation
	// TimeHourAgoOperation left transform time.Time first, then compare with (right - n*time.Hour)
	TimeHourAgoOperation
	// TimeWeekAgoOperation left transform time.Time first, then compare with (right - n*24*time.Hour)
	TimeWeekAgoOperation
	// EmptyOperation left value is empty, empty string, empty map or slice
	EmptyOperation
	// NotEmptyOperation contrary to above
	NotEmptyOperation
	GreaterThan
	LessThan
)

func (c comparator) Compare(operation Operation, left, right any) (bool, error) {
	switch operation {
	case EqualsOperation:
		return assertEquals(left, right)
	case NotEqualsOperation:
		pass, err := assertEquals(left, right)
		if err != nil {
			return false, err
		}
		return !pass, nil
	case ContainsOperation:
		return assertContains(left, right, false)
	case ContainsLowerCasedOperation:
		return assertContains(left, right, true)
	case NotContainsOperation:
		pass, err := assertContains(left, right, false)
		if err != nil {
			return false, err
		}
		return !pass, nil
	case NotContainsLowerCasedOperation:
		pass, err := assertContains(left, right, true)
		if err != nil {
			return false, err
		}
		return !pass, nil
	case StringStartWithOperation:
		return assertStartWith(left, right)
	case StringEndWithOperation:
		return assertEndWith(left, right)
	case TimeBeforeOperation:
		return assertTimeBefore(left, right)
	case TimeAfterOperation:
		return assertTimeAfter(left, right)
	case TimeDayAgoOperation:
		return assertTimeAge(left, right, time.Hour*24)
	case TimeWeekAgoOperation:
		return assertTimeAge(left, right, time.Hour*24*7)
	case TimeHourAgoOperation:
		return assertTimeAge(left, right, time.Hour)
	case EmptyOperation:
		return assertEmpty(left), nil
	case NotEmptyOperation:
		return !assertEmpty(left), nil
	case GreaterThan:
		return assertGreater(left, right)
	case LessThan:
		return assertLess(left, right)
	default:
		panic("not implemented")
	}
}

func assertTimeBefore(left any, right any) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}
	leftTime := toTime(leftStr)
	if leftTime.err != nil {
		return false, fmt.Errorf("left time invalid: %w", err)
	}
	rightTime := toTime(rightStr)
	if rightTime.err != nil {
		return false, fmt.Errorf("right time invalid: %w", err)
	}
	return leftTime.before(rightTime.Time)
}

func assertTimeAfter(left any, right any) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}
	leftTime := toTime(leftStr)
	if leftTime.err != nil {
		return false, fmt.Errorf("left time invalid: %w", err)
	}
	rightTime := toTime(rightStr)
	if rightTime.err != nil {
		return false, fmt.Errorf("right time invalid: %w", err)
	}
	return leftTime.After(rightTime.Time), nil
}

func assertGreater(left, right any) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}

	lv, err := decimal.NewFromString(leftStr)
	if err != nil {
		return false, fmt.Errorf("cast left to decimal: %w", err)
	}

	rv, err := decimal.NewFromString(rightStr)
	if err != nil {
		return false, fmt.Errorf("cast right to decimal: %w", err)
	}

	return lv.GreaterThan(rv), nil
}

func assertLess(left, right any) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}

	lv, err := decimal.NewFromString(leftStr)
	if err != nil {
		return false, fmt.Errorf("cast left to decimal: %w", err)
	}

	rv, err := decimal.NewFromString(rightStr)
	if err != nil {
		return false, fmt.Errorf("cast right to decimal: %w", err)
	}

	return lv.LessThan(rv), nil
}

func assertEmpty(left any) bool {
	leftStr, err := cast.ToStringE(left)
	if err == nil {
		return leftStr == ""
	}
	leftSlice, err := trans.ToAnySlice(left)
	if err == nil {
		return len(leftSlice) == 0
	}
	leftMap, err := cast.ToStringMapE(left)
	if err == nil {
		return len(leftMap) == 0
	}
	return false
}

func assertEndWith(left any, right any) (bool, error) {
	return assertTwoString(left, right, strings.HasSuffix)
}

func assertStartWith(left any, right any) (bool, error) {
	return assertTwoString(left, right, strings.HasPrefix)
}

type compareFn func(left, right string) bool

func assertTwoString(left, right any, fn compareFn) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}
	return fn(leftStr, rightStr), nil
}

func assertEquals(left, right any) (bool, error) {
	return assertTwoString(left, right, func(left, right string) bool {
		return left == right
	})
}

func assertContains(left, right any, lowerCased bool) (bool, error) {
	rightStr, err := cast.ToStringE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to string: %w", err)
	}
	leftStr, err := cast.ToStringE(left)
	if err == nil {
		if lowerCased {
			return strings.Contains(strings.ToLower(leftStr), strings.ToLower(rightStr)), nil
		}

		return strings.Contains(leftStr, rightStr), nil
	}

	leftSlice, err := trans.ToStringSlice(left)
	if err == nil {
		for _, leftItem := range leftSlice {
			if lowerCased {
				if strings.EqualFold(leftItem, rightStr) {
					return true, nil
				}
				continue
			}
			if leftItem == rightStr {
				return true, nil
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("left cannot trans to string or []string: %w", err)
}

func assertTimeAge(left, right any, beforeDuration time.Duration) (bool, error) {
	leftStr, err := cast.ToStringE(left)
	if err != nil {
		return false, fmt.Errorf("cast left to string: %w", err)
	}
	rightInt, err := cast.ToIntE(right)
	if err != nil {
		return false, fmt.Errorf("cast right to int: %w", err)
	}
	return toTime(leftStr).before(time.Now().Add(time.Duration(-rightInt) * beforeDuration))
}

func NewComparator() Comparator {
	return &comparator{}
}
