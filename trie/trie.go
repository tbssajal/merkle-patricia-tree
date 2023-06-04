package trie

import (
	"bytes"
	"errors"
	"fmt"
	"merkle-patricia-tree/storage/version_handler"
	"merkle-patricia-tree/trie/nodes"
	"merkle-patricia-tree/trie/repository"
	"merkle-patricia-tree/utils"
)

// Implementation of persistent merkle-patricia-tree
// according to ethereum yellow paper appendix-D.
// The persistent structure allows to rollback to previous version

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

type VersionHandler = version_handler.VersionHandler

type TrieHashMap struct {
	root 			*nodes.Node
	rootId 			uint64
	nodeCache		map[uint64]*nodes.Node
	versionHandler	*VersionHandler
}

func (trie *TrieHashMap) init(versionHandler *VersionHandler) {
	trie.nodeCache = make(map[uint64]*nodes.Node)
	trie.versionHandler = versionHandler
}

func NewTrieHashMap(versionHandler *VersionHandler) *TrieHashMap {
	trie := new(TrieHashMap)
	trie.init(versionHandler)
	trie.root = nil
	trie.rootId = 0
	return trie
}

func NewTrieHashMapFromRoot(rootId uint64, versionHandler *VersionHandler) *TrieHashMap {
	trie := new(TrieHashMap)
	trie.init(versionHandler)
	root, err := trie.GetNodeById(rootId)
	if err != nil {
		return nil
	}
	trie.root = root
	trie.rootId = rootId
	return trie
}

func (trie *TrieHashMap) Get(key []byte) ([]byte, error) {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		return nil, errors.New("Incorrect hash length")
	}
	return trie.find(key)
}

func (trie *TrieHashMap) Put(key []byte, value []byte) {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		panic("Incorrect hash length")
	}
	rootId, err := trie.addNode(trie.rootId, 0, key, value, true)
	if err == nil {
		root, newErr := trie.GetNodeById(rootId)
		if newErr == nil {
			trie.root = root
			trie.rootId = rootId
		} else {
			fmt.Println("Failed to add (key, value) pair")
		}
	} else {
		fmt.Println("Failed to add (key, value) pair")
	}
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

func (trie *TrieHashMap) addNode(
	rootId uint64, nibbleHeight byte, keyHash []byte, value []byte, checkIfPresent bool,
) (uint64, error) {
	if rootId == 0 {
		return trie.newLeafNode(keyHash, value), nil
	}

	node, err := trie.GetNodeById(rootId);
	if err != nil {
		return 0, err
	}

	if node.Type() == nodes.Leaf {
		if bytes.Equal(keyHash, node.KeyHash()) == false {
			return trie.splitLeafNode(rootId, node, nibbleHeight, keyHash, value), nil
		} else if (checkIfPresent) {
			return 0, errors.New("Key already present")
		} else {
			return trie.updateLeafNode(rootId, node, value), nil
		}
	} else if node.Type() == nodes.Extension {
		nibbleCount := node.NibbleCount()
		nibbles := node.Nibbles()
		matched := 0
		for matched = 0; matched < int(nibbleCount); matched++ {
			extensionNibble := utils.GetNibble(nibbles, matched)
			currentNibble := utils.GetNibble(keyHash, int(nibbleHeight))
			nibbleHeight++
			if extensionNibble != currentNibble {
				break
			}
		}

		if matched == int(nibbleCount) {
			newNodeId, err := trie.addNode(node.Child(), nibbleHeight, keyHash, value, checkIfPresent)
			if err != nil {
				return 0, err
			}

			if newNodeId == node.Child() {
				return rootId, nil
			} else {
				newNode, err := trie.GetNodeById(newNodeId)
				if err != nil {
					panic(err)
				}
				return trie.newExtensionNode(nibbleCount, nibbles, newNodeId, newNode.Hash()), nil
			}
		} else {
			newLeafNodeId := trie.newLeafNode(keyHash, value)
			newLeafNode, err := trie.GetNodeById(newLeafNodeId)
			if err != nil {
				panic(err)
			}

			remainingNibble := nibbleCount - byte(matched)
			oldNodeChildId := node.Child()
			oldNodeChild, err := trie.GetNodeById(oldNodeChildId)
			if err != nil {
				panic(err)
			}

			var newBranchNode *nodes.Node
			if remainingNibble == 1 {
				newBranchNodeId := trie.newBranchNodeFromLeaves(
					oldNodeChildId, newLeafNodeId, nibbles[nibbleCount - 1], utils.GetNibble(keyHash, int(nibbleHeight)),
					oldNodeChild.Hash(), newLeafNode.Hash(),
				)
				newBranchNode, err = trie.GetNodeById(newBranchNodeId)
				if err != nil {
					panic(err)
				}
			}
		}

	}
	// switch (rootNode)
	// {
	// 	case InternalNode internalNode:
	// 		var h = HashFragment(keyHash, height);
	// 		var to = internalNode.GetChildByHash(h);
	// 		var updatedTo = AddInternal(to, height + 1, keyHash, value, check);
	// 		return ModifyInternalNode(root, internalNode, h, updatedTo,
	// 			GetNodeById(updatedTo)?.Hash ?? throw new InvalidOperationException()
	// 		);
	// 	case LeafNode leafNode:
	// 		if (!leafNode.KeyHash.SequenceEqual(keyHash))
	// 			return SplitLeafNode(root, leafNode, height, keyHash, value);
	// 		if (check)
	// 			throw new ArgumentException("Specified keyHash is already present or hash collision occured");
	// 		return UpdateLeafNode(root, leafNode, value);
	// }
}

func (trie *TrieHashMap) splitLeafNode(
	oldNodeId uint64, oldNode *nodes.Node, nibbleHeight byte, keyHash []byte, value []byte,
) uint64 {
	oldNodeKeyHash := oldNode.KeyHash()
	newNodeId := trie.newLeafNode(keyHash, value)
	newNode, err := trie.GetNodeById(newNodeId)
	if err != nil {
		panic(err)
	}

	var nibbleCount byte = 0
	var oldNodeNibble byte
	var newNodeNibble byte
	var nibbles []byte
	for nibbleHeight < TrieMaxHeight {
		oldNodeNibble = utils.GetNibble(oldNodeKeyHash, int(nibbleHeight))
		newNodeNibble = utils.GetNibble(keyHash, int(nibbleHeight))
		if newNodeNibble != oldNodeNibble {
			break
		}
		nibbles = append(nibbles, oldNodeNibble)
		nibbleCount++
		nibbleHeight++
	}

	if nibbleHeight >= TrieMaxHeight {
		panic("Cannot split, both leaf nodes have same key")
	}

	newBranchId := trie.newBranchNodeFromLeaves(
		oldNodeId, newNodeId, oldNodeNibble, newNodeNibble, oldNode.Hash(), newNode.Hash(),
	)
	if nibbleCount == 0 {
		return newBranchId
	}

	newBranchNode, err := trie.GetNodeById(newBranchId)
	if err != nil {
		panic(err)
	}

	if nibbleCount == 1 {
		return trie.newBranchNode(1<<nibbles[0], []uint64 {newBranchId}, [][]byte {newBranchNode.Hash()})
	} else {
		return trie.newExtensionNode(
			nibbleCount, utils.NibblesToBytes(nibbles, int(nibbleCount)), newBranchId, newBranchNode.Hash(),
		)
	}
}

func (trie *TrieHashMap) newBranchNodeFromLeaves(
	oldNodeId uint64, newNodeId uint64, oldNodeNibble byte, newNodeNibble byte, oldNodeHash []byte, newNodeHash []byte,
) uint64 {
	var childIds []uint64
	var childHashes [][]byte
	if oldNodeNibble < newNodeNibble {
		childIds = append(childIds, oldNodeId)
		childIds = append(childIds, newNodeId)

		childHashes = append(childHashes, oldNodeHash)
		childHashes = append(childHashes, newNodeHash)
	} else {
		childIds = append(childIds, newNodeId)
		childIds = append(childIds, oldNodeId)

		childHashes = append(childHashes, newNodeHash)
		childHashes = append(childHashes, oldNodeHash)
	}

	return trie.newBranchNode(uint16((1<<oldNodeNibble) | (1<<newNodeNibble)), childIds, childHashes)
}

func (trie *TrieHashMap) newBranchNode(mask uint16, childIds []uint64, childHashes [][]byte) uint64 {
	newNode, err := nodes.NewBranchNode(mask,  childIds, childHashes)
	if err != nil {
		panic(err)
	}
	return trie.saveNode(newNode)
}

func (trie *TrieHashMap) newExtensionNode(nibbleCount byte, nibbles []byte, child uint64, childNodeHash []byte) uint64 {
	newNode, err := nodes.NewExtensionNode(nibbleCount, nibbles, child, childNodeHash)
	if err != nil {
		panic(err)
	}
	return trie.saveNode(newNode)
}

func (trie *TrieHashMap) newLeafNode(keyHash []byte, value []byte) uint64 {
	newNode, err := nodes.NewLeafNode(keyHash, value)
	if err != nil {
		panic(err)
	}
	return trie.saveNode(newNode)
}

func (trie *TrieHashMap) updateLeafNode(nodeId uint64, node *nodes.Node, value []byte) uint64 {
	if bytes.Equal(value, node.Value()) {
		return nodeId
	} else {
		return trie.newLeafNode(node.KeyHash(), value)
	}
}

func (trie *TrieHashMap) saveNode(newNode *nodes.Node) uint64 {
	newNodeId := trie.versionHandler.NewVersion()
	trie.nodeCache[newNodeId] = newNode
	return newNodeId
}

func (trie *TrieHashMap) GetNodeById(nodeId uint64) (*nodes.Node, error) {
	node, ok := trie.nodeCache[nodeId]
	if ok == false {
		return repository.GetNodeById(nodeId)
	} else {
		return node, nil
	}
}