package trie

import (
	"bytes"
	"errors"
	"merkle-patricia-tree/trie/nodes"
	"merkle-patricia-tree/trie/repository"
	"merkle-patricia-tree/utils"
)

type iTrie interface {
	// Get returns the value associate with the key
	// error is returned if the key is not found.
	Get(key []byte) ([]byte, error)
	// Put inserts the [key,value] node in the trie
	Put(key []byte, value []byte)
	// Del removes a node from the trie
	// returns an error if not found.
	Del(key []byte) error
	// Commit saves the trie in persistent storage
	// and returns the trie root key.
	Commit() []byte
	// Proof returns the Merkle-proof associated with
	// a node. An error is returned if the node is not found.
	Proof(key []byte) ([][]byte, error)
}

const (
	TrieMaxHeight = 64
)

type TrieHashMap struct {
	root 		*nodes.Node
	rootId 		uint64
	nodeCache	map[uint64]*nodes.Node
}

func NewTrieHashMap() *TrieHashMap {
	trie := new(TrieHashMap)
	trie.root = nil
	trie.rootId = 0
	trie.nodeCache = make(map[uint64]*nodes.Node)
	return trie
}

func (trie *TrieHashMap) Get(key []byte) ([]byte, error) {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		return nil, errors.New("Incorrect hash length")
	}
	return trie.find(key)
}

func (trie *TrieHashMap) find(key []byte) ([]byte, error) {
	currentNodeId := trie.rootId
	nibbleHeight := 0
	for nibbleHeight < TrieMaxHeight {
		if currentNodeId == 0 {
			return nil, errors.New("Key not found")
		}
		currentNode, error := trie.GetNodeById(currentNodeId);
		if error != nil {
			return nil, error
		}

		if currentNode.Type() == nodes.Leaf {
			if bytes.Equal(key, currentNode.KeyHash()) {
				return currentNode.Value(), nil
			} else {
				return nil, errors.New("Key not found")
			}
		} else if currentNode.Type() == nodes.Extension {
			nibbleCount := int(currentNode.NibbleCount())
			if bytes.Equal(
				currentNode.Nibbles(), utils.GetNibbles(key, nibbleHeight, nibbleCount),
			) {
				currentNodeId = currentNode.Child()
				nibbleHeight += nibbleCount
			} else {
				return nil, errors.New("Key not found")
			}
		} else if currentNode.Type() == nodes.Branch {
			nibble := utils.GetNibble(key, nibbleHeight)
			currentNodeId = currentNode.GetChildByNibble(nibble)
			nibbleHeight++
		} else {
			return nil, errors.New("Unsupported node type")
		}
	}

	return nil, errors.New("Key not found")
}

func (trie *TrieHashMap) GetNodeById(nodeId uint64) (*nodes.Node, error) {
	node, ok := trie.nodeCache[nodeId]
	if ok == false {
		return repository.GetNodeById(nodeId)
	} else {
		return node, nil
	}
}