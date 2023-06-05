package storage

import "merkle-patricia-tree/utils"

const (
	// nodes
	NodeById     byte = 0x01
	NodeIdByHash byte = 0x02

	// version
	LastVersion byte = 0x03
)

func BuildPrefix(prefix byte, key []byte) []byte {
	var keyWithPrefix []byte
	keyWithPrefix = append(keyWithPrefix, prefix)
	keyWithPrefix = append(keyWithPrefix, key...)
	return keyWithPrefix
}

func PrefixNodeById(id uint64) []byte {
	return BuildPrefix(NodeById, utils.UInt64ToBytes(id))
}

func PrefixNodeIdByHash(hash []byte) []byte {
	return BuildPrefix(NodeIdByHash, hash)
}