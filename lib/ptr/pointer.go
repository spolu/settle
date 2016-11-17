package ptr

// True returns a pointer to a true bool
func True() *bool {
	ret := true
	return &ret
}

// False returns a pointer to a false bool
func False() *bool {
	ret := false
	return &ret
}

// Str returns a pointer to a string
func Str(str string) *string {
	ret := str
	return &ret
}

// DerefStr returns `*strPtr` if `strPtr` is non-nil.
// Otherwise it returns `defaultValue`.
func DerefStr(strPtr *string, defaultValue string) string {
	if strPtr != nil {
		return *strPtr
	}
	return defaultValue
}

// Int64 returns a pointer to an int64
func Int64(n int64) *int64 {
	ret := n
	return &ret
}

// Int returns a pointer to an int
func Int(n int) *int {
	ret := n
	return &ret
}

// Float64 returns a pointer to a float64
func Float64(n float64) *float64 {
	ret := n
	return &ret
}
