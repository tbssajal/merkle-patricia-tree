package utils

import "encoding/binary"

func UInt64ToBytes(value uint64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, value)
	return res
}

func UInt64FromBytes(value []byte) uint64 {
	return binary.LittleEndian.Uint64(value)
}

func UInt16ToBytes(value uint16) []byte {
	res := make([]byte, 2)
	binary.LittleEndian.PutUint16(res, value)
	return res
}

func UInt16FromBytes(value []byte) uint16 {
	return binary.LittleEndian.Uint16(value)
}