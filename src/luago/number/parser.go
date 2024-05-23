package number

import "strconv"

func ParseInteger(s string) (int64, bool) {
	i, err := strconv.ParseInt(s, 10, 64)
	return i, err == nil
}

func ParseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}
