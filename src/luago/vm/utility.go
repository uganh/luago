package vm

/**
 * Converts an integer to a "floating point byte", represented as (eeeeexxx),
 * where the real value is (1xxx) * 2 ^ (eeeee - 1) if eeeee !=0 and (xxx)
 * otherwise.
 */
func Int2FPB(x int) int {
	if x < 8 {
		return x
	}
	e := 0              // exponent
	for x >= (8 << 4) { // coarse steps
		x = (x + 0xf) >> 4 // x = ceil(x / 16)
		e += 4
	}
	for x >= (8 << 1) { // fin steps
		x = (x + 1) >> 1 // x = ceil(x / 2)
		e++
	}
	return ((e + 1) << 3) | (x - 8)
}

/**
 * Converts back
 */
func FPB2Int(x int) int {
	if x < 8 {
		return x
	} else {
		return ((x & 7) + 8) << uint((x>>3)-1)
	}
}
