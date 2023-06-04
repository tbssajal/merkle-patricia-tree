package utils

import "fmt"

func Test() {
	fmt.Println("testing");
}

func GetNibbles(data []byte, startPos int, nibbleCount int) []byte{
	nibbles := make([]byte, nibbleCount)
	dataLen := len(data)
	for i := 0; i < nibbleCount && startPos < dataLen; i++ {
		if (startPos & 1) != 0 {
			nibbles[i] = data[startPos >> 1] & 15 // nibbles[i] = data[startPos / 2] % 16
		} else {
			nibbles[i] = data[startPos >> 1] >> 4 // nibbles[i] = data[startPos / 2] / 16
		}
		startPos++
	}

	return NibblesToBytes(nibbles, nibbleCount)
}

func NibblesToBytes(nibbles []byte, nibbleCount int) []byte {
	nibblesInBytes := make([]byte, (nibbleCount + 1) >> 1)
	for i := 0; i < nibbleCount; i++ {
		if (i & 1) != 0 {
			nibblesInBytes[i >> 1] |= nibbles[i]
		} else {
			nibblesInBytes[i >> 1] = nibbles[i] << 4
		}
	}
	return nibblesInBytes
}

func GetNibble(data []byte, nibbleHeight int) byte {
	if (nibbleHeight >> 1) >= len(data) {
		return 0
	}

	if (nibbleHeight & 1) != 0 {
		return data[nibbleHeight >> 1] & 15 // nibble = data[nibbleHeight / 2] % 16
	} else {
		return data[nibbleHeight >> 1] >> 4 // nibble = data[nibbleHeight / 2] / 16
	}
}

func PositionOf(mask uint16, h byte) byte {
	var res byte = 0
	for h > 0 {
		if (mask & 1) != 0 {
			res++
		}
		h--
		mask >>= 1
	}
	return res
}