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

// TODO: implement serializer
func (node *Node) LeafNodeToBytes() []byte {
	
}

func LeafNodeFromBytes(bytes []byte) (*Node, error) {
	
}