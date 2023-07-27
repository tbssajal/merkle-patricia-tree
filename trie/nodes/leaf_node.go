package nodes

import (
	"errors"
	"merkle-patricia-tree/utils"
)

func NewLeafNode(keyHash []byte, value []byte) (*Node, error) {
	if len(keyHash) != HashLen {
		return nil, errors.New("Key hash has incorrect length")
	}
	node := new(Node)
	node.nodeType = Leaf
	node.keyHash = make([]byte, HashLen)
	copy(node.keyHash, keyHash)
	node.value = make([]byte, len(value))
	copy(node.value, value)
	hash, err := ComputeLeafNodeHash(node.KeyHash(), node.Value())
	if err != nil {
		return nil, err
	}
	node.hash = make([]byte, HashLen)
	copy(node.hash, hash)
	return node, nil
}

func (node *Node) KeyHash() []byte {
	keyHash := make([]byte, HashLen)
	copy(keyHash, node.keyHash)
	return keyHash
}

func (node *Node) Value() []byte {
	value := make([]byte, len(node.value))
	copy(value, node.value)
	return value
}

func ComputeLeafNodeHash(keyHash []byte, value []byte) ([]byte, error) {
	var hash []byte
	hash = append(hash, keyHash...)
	hash = append(hash, value...)
	hash = utils.Keccak256(hash)
	if len(hash) != HashLen {
		return nil, errors.New("node hash has incorrect length")
	}
	return hash, nil
}

func (node *Node) LeafNodeToBytes() []byte {
	nodeTypeLen := 1
	keyHashLen := HashLen
	valueLen := len(node.value)
	bytes := make([]byte, nodeTypeLen + keyHashLen + valueLen)

	pos := 0
	bytes[pos] = byte(node.nodeType)
	pos += nodeTypeLen

	copy(bytes[pos:pos + keyHashLen], node.keyHash)
	pos += keyHashLen

	copy(bytes[pos:pos+valueLen], node.value)
	return bytes
}

func LeafNodeFromBytes(bytes []byte) (*Node, error) {
	nodeTypeLen := 1
	keyHashLen := HashLen
	if (len(bytes) < nodeTypeLen + keyHashLen) {
		return nil, errors.New("Not enough bytes to decode leaf node")
	}

	pos := 0
	nodeType := bytes[pos]
	if nodeType != byte(Leaf) {
		return nil, errors.New("Cannot decode leaf node, incorrect node type")
	}
	pos += nodeTypeLen

	keyHash := make([]byte, keyHashLen)
	copy(keyHash, bytes[pos:pos+keyHashLen])
	pos += keyHashLen

	var value []byte
	value = append(value, bytes[pos:]...)
	return NewLeafNode(keyHash, value)
}