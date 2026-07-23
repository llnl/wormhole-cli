package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// StringOrInt is a custom type that handles JSON Unmarshal for string or int values.
type StringOrInt string

func (s *StringOrInt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrInt(str)
		return nil
	}

	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		*s = StringOrInt(strconv.Itoa(intValue))
		return nil
	}

	return fmt.Errorf("StringOrInt: unable to unmarshal %s", string(data))
}

func ToStringOrInt(i any) (StringOrInt, error) {
	switch v := i.(type) {
	case string:
		return StringOrInt(v), nil
	case int:
		return StringOrInt(strconv.Itoa(v)), nil
	case int64:
		return StringOrInt(strconv.FormatInt(v, 10)), nil
	default:
		return "", errors.New("ToStringOrInt: unsupported type")
	}
}
