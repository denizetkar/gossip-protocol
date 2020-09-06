package mathutils

// Min returns the minimum of x and y, preferring x.
func Min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

// Max returns the maximum of x and y, preferring x.
func Max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}
