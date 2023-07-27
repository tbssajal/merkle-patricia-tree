package utils

func PositionOf(mask uint16, h byte) byte {
	var res byte = 0
	var bit uint16 = 1 << h
	for mask > 0 {
		nextMask := mask & (mask - 1)
		if (mask ^ nextMask) >= bit {
			return res
		}
		mask = nextMask
		res++
	}
	return res
}