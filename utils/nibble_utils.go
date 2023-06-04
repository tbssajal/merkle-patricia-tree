package utils

func GetNibbles(data []byte, startPos int, nibbleCount int) []byte {
	nibbles := make([]byte, nibbleCount)
	dataLen := len(data)
	for i := 0; i < nibbleCount && startPos < dataLen; i++ {
		if (startPos & 1) != 0 {
			nibbles[i] = data[startPos>>1] & 15 // nibbles[i] = data[startPos / 2] % 16
		} else {
			nibbles[i] = data[startPos>>1] >> 4 // nibbles[i] = data[startPos / 2] / 16
		}
		startPos++
	}

	return NibblesToBytes(nibbles, nibbleCount)
}

func NibblesToBytes(nibbles []byte, nibbleCount int) []byte {
	nibblesInBytes := make([]byte, (nibbleCount+1)>>1)
	for i := 0; i < nibbleCount; i++ {
		if (i & 1) != 0 {
			nibblesInBytes[i>>1] |= nibbles[i]
		} else {
			nibblesInBytes[i>>1] = nibbles[i] << 4
		}
	}
	return nibblesInBytes
}

func GetNibble(data []byte, nibbleHeight int) byte {
	if (nibbleHeight >> 1) >= len(data) {
		return 0
	}

	if (nibbleHeight & 1) != 0 {
		return data[nibbleHeight>>1] & 15 // nibble = data[nibbleHeight / 2] % 16
	} else {
		return data[nibbleHeight>>1] >> 4 // nibble = data[nibbleHeight / 2] / 16
	}
}

func ExtendNibblesByOne(nibbles []byte, nibblesCount int, extraNibble byte) []byte {
	var res []byte
	res = append(res, nibbles...)
	if (nibblesCount & 1) == 0 {
		res = append(res, extraNibble<<4)
	} else {
		res[nibblesCount>>1] |= extraNibble
	}
	return res
}

func MergeNibbles(nibbles []byte, nibblesCount int, extraNibbles []byte, extraNibbleCount int) []byte {
	var res []byte
	res = append(res, nibbles...)
	if (nibblesCount & 1) == 0 {
		// even number of nibbles in first array, so we can just append extraNibbles at the end
		res = append(res, extraNibbles...)
	} else {
		// we need to merge the first nibble of extraNibbles with the last nibble of nibbles
		// then we can append the rest of extraNibbles at the end
		res[nibblesCount>>1] |= GetNibble(extraNibbles, 0)
		res = append(res, GetNibbles(extraNibbles, 1, extraNibbleCount-1)...)
	}
	return res
}