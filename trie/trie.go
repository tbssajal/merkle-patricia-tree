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

// TODO: add tests

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
	// and returns the trie root hash.
	Commit() []byte
	// Proof returns the Merkle-proof associated with
	// a node. An error is returned if the node is not found.
	Proof(key []byte) ([][]byte, error)
	// Verifies the proof related with the key
	VerifyProof(key []byte, proof [][]byte) bool
}

const (
	TrieMaxHeight = 64
)

type VersionHandler = version_handler.VersionHandler
type Repository = repository.NodeRepository

type TrieHashMap struct {
	root           *nodes.Node
	rootId         uint64
	nodeCache      map[uint64]*nodes.Node
	versionHandler *VersionHandler
	nodeRepo       *repository.NodeRepository
	EmptyHash      []byte
}

func (trie *TrieHashMap) init(versionHandler *VersionHandler, nodeRepo *Repository) {
	trie.nodeCache = make(map[uint64]*nodes.Node)
	trie.versionHandler = versionHandler
	trie.nodeRepo = nodeRepo
	trie.EmptyHash = make([]byte, nodes.HashLen)
}

func NewTrieHashMap(versionHandler *VersionHandler, nodeRepo *Repository) *TrieHashMap {
	trie := new(TrieHashMap)
	trie.init(versionHandler, nodeRepo)
	trie.root = nil
	trie.rootId = 0
	return trie
}

func NewTrieHashMapFromRoot(rootId uint64, versionHandler *VersionHandler, nodeRepo *Repository) *TrieHashMap {
	trie := new(TrieHashMap)
	trie.init(versionHandler, nodeRepo)
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
		if rootId == trie.rootId {
			return
		}
		root, newErr := trie.GetNodeById(rootId)
		if newErr == nil {
			trie.root = root
			trie.rootId = rootId
		} else {
			fmt.Println("Failed to fetch node with id", rootId)
			panic("Failed to fetch node")
		}
	} else {
		fmt.Println("Failed to add (key, value) pair, error:", err)
	}
}

func (trie *TrieHashMap) Del(key []byte) error {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		panic("Incorrect hash length")
	}
	rootId, err := trie.deleteNode(trie.rootId, 0, key)
	if err == nil {
		if rootId == trie.rootId {
			return nil
		}

		if rootId == 0 {
			trie.rootId = 0
			trie.root = nil
			return nil
		}

		root, err := trie.GetNodeById(rootId)
		if err == nil {
			trie.root = root
			trie.rootId = rootId
			return nil
		} else {
			fmt.Println("Failed to fetch node with id", rootId)
			panic("Failed to fetch node")
		}
	} else {
		return err
	}
}

func (trie *TrieHashMap) Commit() []byte {
	if trie.rootId == 0 {
		return trie.EmptyHash
	} else {
		trie.persistNodes(trie.rootId)
		return trie.root.Hash()
	}
}

// in proof we put:
// proof[0] : value mapped with key at leaf node; leaf node hash : keccak256(keyHash.value)
// proof[-1] : root hash
// remaining elements of proof help to calculate the root hash
func (trie *TrieHashMap) Proof(key []byte) ([][]byte, error) {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		panic("Incorrect hash length")
	}

	var proof [][]byte
	err := trie.findProof(trie.rootId, 0, key, proof)
	if err != nil {
		return nil, err
	} else {
		proof = append(proof, trie.root.Hash())
		return proof, nil
	}
}

// Verifies the proof related with the key
func (trie *TrieHashMap) VerifyProof(key []byte, proof [][]byte) bool {
	key = utils.Keccak256(key)
	if len(key) != nodes.HashLen {
		panic(errors.New("Incorrect hash length"))
	}
	proofLen := len(proof)
	if proofLen < 2 {
		return false
	}

	var nibbleHeight byte = 0
	for i := 1; i < proofLen-1; i++ {
		proofSegmentLen := len(proof[i])
		if proofSegmentLen == 1 {
			// extension node
			nibbleHeight += proof[i][0]
		} else if proofSegmentLen > 1 {
			// branch node
			nibbleHeight++
		} else {
			// invalid
			return false
		}
	}

	// we have value mapped with key at proof[0]
	// we calculate leaf node hash with it
	// remaning elements help calculate hash of each parent node
	// finally we get hash of root
	hash, err := nodes.ComputeLeafNodeHash(key, proof[0])
	if err != nil {
		return false
	}

	for i := 1; i < proofLen-1; i++ {
		proofSegmentLen := len(proof[i])
		if proofSegmentLen == 1 {
			// extension node
			nibbleHeight -= proof[i][0]
			nibbles := utils.GetNibbles(key, int(nibbleHeight), int(proof[i][0]))
			// calucalate parent hash
			hash, err = nodes.ComputeExtensionNodeHash((proof[i][0]&1) != 0, nibbles, hash)
			if err != nil {
				return false
			}
		} else if proofSegmentLen > 1 {
			// branch node
			nibbleHeight--
			mask := uint16(proof[i][0])
			nibble := utils.GetNibble(key, int(nibbleHeight))
			if (mask & (1 << nibble)) == 0 {
				return false
			}
			pos := utils.PositionOf(mask, nibble)
			var childHashes [][]byte

			proofSegmentLen := len(proof[i])
			for i := 1; i < proofSegmentLen; i += nodes.HashLen {
				if i+nodes.HashLen > proofSegmentLen {
					return false
				}
				childHashes = append(childHashes, proof[i][i:i+nodes.HashLen])
			}
			// calculate parent hash
			parHash, err := nodes.ComputeBranchNodeHash(mask, childHashes)
			if err != nil {
				return false
			}
			// check if childHashes contain my hash
			if !bytes.Equal(hash, childHashes[pos]) {
				return false
			}
			hash = parHash
		} else {
			// invalid
			return false
		}
	}

	// check if root hash equals to calculated hash
	return bytes.Equal(hash, proof[proofLen-1])
}

func (trie *TrieHashMap) persistNodes(rootId uint64) {
	node, ok := trie.nodeCache[rootId]
	if ok == false {
		return
	}
	var childIds []uint64
	if node.Type() == nodes.Branch {
		childIds = node.Children()
	} else if node.Type() == nodes.Extension {
		childIds = append(childIds, node.Child())
	}

	childCount := len(childIds)
	for i := 0; i < childCount; i++ {
		trie.persistNodes(childIds[i])
	}
	trie.nodeRepo.SaveNode(rootId, node)
	delete(trie.nodeCache, rootId)
}

func (trie *TrieHashMap) deleteNode(rootId uint64, nibbleHeight byte, keyHash []byte) (uint64, error) {
	if rootId == 0 {
		return 0, errors.New("Key not found")
	}

	node, err := trie.GetNodeById(rootId)
	if err != nil {
		return 0, err
	}

	if node.Type() == nodes.Leaf {
		if bytes.Equal(keyHash, node.KeyHash()) {
			return 0, nil
		} else {
			return 0, errors.New("Key not found")
		}
	} else if node.Type() == nodes.Extension {
		nibbleCount := node.NibbleCount()
		if bytes.Equal(node.Nibbles(), utils.GetNibbles(keyHash, int(nibbleHeight), int(nibbleCount))) {
			newNodeId, err := trie.deleteNode(node.Child(), nibbleHeight+nibbleCount, keyHash)
			if err != nil {
				return 0, err
			} else {
				// key is deleted
				// current node is extension node
				// that means its child must be a branch node with at least 2 children
				// but we just deleted a key, so there is a possibility that the child is not a branch anymore
				// it could be an extension or a leaf
				// if the child is not a branch, current extension will be merged with its child
				return trie.updateOrShrinkExtension(node, newNodeId), nil
			}
		} else {
			return 0, errors.New("Key not found")
		}
	} else if node.Type() == nodes.Branch {
		nibble := utils.GetNibble(keyHash, int(nibbleHeight))
		newNodeId, err := trie.deleteNode(node.GetChildByNibble(nibble), nibbleHeight+1, keyHash)
		if err != nil {
			return 0, err
		} else {
			return trie.updateOrShrinkBranch(node, newNodeId, nibble), nil
		}
	} else {
		return 0, errors.New("Unsupported node type")
	}
}

func (trie *TrieHashMap) updateOrShrinkBranch(branchNode *nodes.Node, newNodeId uint64, nibble byte) uint64 {
	var childId uint64
	var childNibble byte
	childrenCount := branchNode.ChildrenCount()
	if newNodeId == 0 {
		if childrenCount == 1 {
			panic("Impossible")
		} else if childrenCount == 2 {
			childNibbles := branchNode.GetChildNibbles()
			for i := 0; i < int(childrenCount); i++ {
				if childNibbles[i] != nibble {
					childId = branchNode.GetChildByNibble(childNibbles[i])
					childNibble = childNibbles[i]
				}
			}
		} else {
			// after deleting key, the node still have more than one children, so it is a valid branch
			// just need to remove deleted child
			return trie.reduceTrieBranchByOne(branchNode, nibble)
		}
	} else if childrenCount > 1 {
		// no child is deleted, but one child is updated and the node has more than one child
		// so it is a valid branch
		return trie.updateTrieBranch(branchNode, newNodeId, nibble)
	} else {
		childId = newNodeId
		childNibble = nibble
	}

	// currently the branch has only one child
	// the only condition for the branch to be valid: the child has to be a branch node with more than one child
	child, err := trie.GetNodeById(childId)
	if err != nil {
		panic(err)
	}

	if child.Type() == nodes.Branch {
		if child.ChildrenCount() > 1 {
			return trie.newBranchNode(1<<childNibble, []uint64{childId}, [][]byte{child.Hash()})
		} else {
			// if the child is a branch with a single child, they will merge and become an extension
			extensionNibble := (childNibble << 4) | child.GetChildNibbles()[0]
			grandChildId := child.Children()[0]
			// the grandChild must be a branch with more than one child
			// so the merge is correct
			grandChild, err := trie.GetNodeById(grandChildId)
			if err != nil {
				panic(err)
			}
			return trie.newExtensionNode(2, []byte{extensionNibble}, grandChildId, grandChild.Hash())
		}
	} else if child.Type() == nodes.Extension {
		// the branch and the extension will merge and become one extension
		extensionNibbles := utils.MergeNibbles([]byte{childNibble}, 1, child.Nibbles(), int(child.NibbleCount()))
		grandChildId := child.Child()
		grandChild, err := trie.GetNodeById(grandChildId)
		if err != nil {
			panic(err)
		}
		return trie.newExtensionNode(1+child.NibbleCount(), extensionNibbles, grandChildId, grandChild.Hash())
	} else if child.Type() == nodes.Leaf {
		// branch and leaf will merge and become leaf
		return childId
	} else {
		panic(errors.New("Unsupported node type"))
	}
}

func (trie *TrieHashMap) updateOrShrinkExtension(extensionNode *nodes.Node, newNodeId uint64) uint64 {
	newNode, err := trie.GetNodeById(newNodeId)
	if err != nil {
		panic(err)
	}

	if newNode.Type() == nodes.Branch {
		// we still have a branch, but as it is a child of extension, it must have at least 2 children
		if newNode.ChildrenCount() > 1 {
			return trie.newExtensionNode(extensionNode.NibbleCount(), extensionNode.Nibbles(), newNodeId, newNode.Hash())
		} else {
			// the branch has only 1 child, so this branch needs to be replaced
			childNodeId := newNode.Children()[0]
			childNode, err := trie.GetNodeById(childNodeId)
			if err != nil {
				panic(err)
			}

			if childNode.Type() == nodes.Leaf {
				// this branch and the current extension will be replaced by the leaf
				return childNodeId
			} else if childNode.Type() == nodes.Branch {
				// the branch will be replaced by its child branch
				// consecutive 2 branch node both cannot have 1 child only
				// as the parent branch node has 1 child, the child branch node must have more than 1 child
				// so the replacement is valid
				nibblesCount := extensionNode.NibbleCount()
				// we need to add one more nibble to the extension node
				nibbles := utils.ExtendNibblesByOne(extensionNode.Nibbles(), int(nibblesCount), newNode.GetChildNibbles()[0])
				return trie.newExtensionNode(nibblesCount+1, nibbles, childNodeId, childNode.Hash())
			} else if childNode.Type() == nodes.Extension {
				// this extension node must have a child branch node and the child branch node must have
				// at least 2 children
				// so the current extension + its child branch node + this extension will become one single extension
				nibblesCount := extensionNode.NibbleCount()
				nibbles := utils.ExtendNibblesByOne(extensionNode.Nibbles(), int(nibblesCount), newNode.GetChildNibbles()[0])
				nibbles = utils.MergeNibbles(nibbles, int(nibblesCount+1), childNode.Nibbles(), int(childNode.NibbleCount()))
				grandChildId := childNode.Child()
				grandChild, err := trie.GetNodeById(grandChildId)
				if err != nil {
					panic(err)
				}
				return trie.newExtensionNode(nibblesCount+1+childNode.NibbleCount(), nibbles, grandChildId, grandChild.Hash())
			} else {
				panic(errors.New("Unsupported node type"))
			}
		}
	} else if newNode.Type() == nodes.Leaf {
		// extension cannot have leaf as child
		return newNodeId
	} else if newNode.Type() == nodes.Extension {
		// extension node cannot have extension as child
		// the two extensions will be merged into one
		nibbles := utils.MergeNibbles(
			extensionNode.Nibbles(), int(extensionNode.NibbleCount()), newNode.Nibbles(), int(newNode.NibbleCount()),
		)
		childId := newNode.Child()
		child, err := trie.GetNodeById(childId)
		if err != nil {
			panic(err)
		}
		return trie.newExtensionNode(extensionNode.NibbleCount()+newNode.NibbleCount(), nibbles, childId, child.Hash())
	} else {
		panic(errors.New("Unsupported node type"))
	}
}

func (trie *TrieHashMap) find(key []byte) ([]byte, error) {
	currentNodeId := trie.rootId
	nibbleHeight := 0
	for nibbleHeight < TrieMaxHeight {
		if currentNodeId == 0 {
			return nil, errors.New("Key not found")
		}
		currentNode, error := trie.GetNodeById(currentNodeId)
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

func (trie *TrieHashMap) findProof(rootId uint64, nibbleHeight byte, keyHash []byte, proof [][]byte) error {
	if rootId == 0 {
		return errors.New("Key not found")
	}

	node, err := trie.GetNodeById(rootId)
	if err != nil {
		return err
	}

	if node.Type() == nodes.Leaf {
		if bytes.Equal(keyHash, node.KeyHash()) {
			proof = append(proof, node.Value())
			return nil
		} else {
			return errors.New("Key not found")
		}
	} else if node.Type() == nodes.Extension {
		nibbleCount := node.NibbleCount()
		if bytes.Equal(
			node.Nibbles(), utils.GetNibbles(keyHash, int(nibbleHeight), int(nibbleCount)),
		) {
			err := trie.findProof(node.Child(), nibbleHeight+nibbleCount, keyHash, proof)
			if err != nil {
				return err
			}
			proof = append(proof, []byte{nibbleCount})
			return nil
		} else {
			return errors.New("Key not found")
		}
	} else if node.Type() == nodes.Branch {
		nibble := utils.GetNibble(keyHash, int(nibbleHeight))
		err := trie.findProof(node.GetChildByNibble(nibble), nibbleHeight+1, keyHash, proof)
		if err != nil {
			return err
		}
		var branchProof []byte
		branchProof = append(branchProof, byte(node.Mask()))

		childIds := node.Children()
		childCount := len(childIds)
		for i := 0; i < childCount; i++ {
			child, err := trie.GetNodeById(childIds[i])
			if err != nil {
				panic(err)
			}
			branchProof = append(branchProof, child.Hash()...)
		}

		return nil
	} else {
		panic("unsupported node type")
	}
}

func (trie *TrieHashMap) addNode(
	rootId uint64, nibbleHeight byte, keyHash []byte, value []byte, checkIfPresent bool,
) (uint64, error) {
	if rootId == 0 {
		return trie.newLeafNode(keyHash, value), nil
	}

	node, err := trie.GetNodeById(rootId)
	if err != nil {
		return 0, err
	}

	if node.Type() == nodes.Leaf {
		if bytes.Equal(keyHash, node.KeyHash()) == false {
			return trie.splitLeafNode(rootId, node, nibbleHeight, keyHash, value), nil
		} else if checkIfPresent {
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
			currentNibble := utils.GetNibble(keyHash, int(nibbleHeight)+matched)
			if extensionNibble != currentNibble {
				break
			}
		}

		// child of extension node must be a branch with at least 2 children
		if matched == int(nibbleCount) {
			newNodeId, err := trie.addNode(
				node.Child(), nibbleHeight+byte(matched), keyHash, value, checkIfPresent,
			)
			if err != nil {
				return 0, err
			}

			if newNodeId == node.Child() {
				return rootId, nil
			} else {
				// as node.Child() is a branch with at least 2 children; newNode must be a branch
				// with at least 2 children because we are adding nodes
				newNode, err := trie.GetNodeById(newNodeId)
				if err != nil {
					panic(err)
				}
				return trie.newExtensionNode(nibbleCount, nibbles, newNodeId, newNode.Hash()), nil
			}
		} else {
			return trie.updateTrieExtension(node, byte(matched), nibbleHeight, keyHash, value), nil
		}

	} else if node.Type() == nodes.Branch {
		nibble := utils.GetNibble(keyHash, int(nibbleHeight))
		childNodeId := node.GetChildByNibble(nibble)
		newNodeId, err := trie.addNode(childNodeId, nibbleHeight+1, keyHash, value, checkIfPresent)
		if err != nil {
			return 0, err
		}

		if newNodeId == childNodeId {
			// no update in the trie
			return rootId, nil
		} else {
			return trie.updateTrieBranch(node, newNodeId, nibble), nil
		}
	} else {
		panic("Unsupported node type")
	}
}

func (trie *TrieHashMap) reduceTrieBranchByOne(branchNode *nodes.Node, nibble byte) uint64 {
	mask := branchNode.Mask()
	childIds := branchNode.Children()
	childCount := len(childIds)
	var newChildHashes [][]byte
	var newChildIds []uint64
	var newNibbleMask uint16 = 1 << nibble
	newMask := mask ^ newNibbleMask

	currentMask := mask
	for i := 0; i < childCount; i++ {
		// getting lowest nibble-mask
		// removing lowest nibble-mask from next-mask
		nextMask := currentMask & (currentMask - 1)
		nibbleMask := currentMask ^ nextMask
		currentMask = nextMask

		if nibbleMask == newNibbleMask {
			continue
		}

		child, err := trie.GetNodeById(childIds[i])
		if err != nil {
			panic(err)
		}
		newChildIds = append(newChildIds, childIds[i])
		newChildHashes = append(newChildHashes, child.Hash())
	}

	return trie.newBranchNode(newMask, newChildIds, newChildHashes)
}

func (trie *TrieHashMap) updateTrieBranch(
	branchNode *nodes.Node, newNodeId uint64, newNodeNibble byte,
) uint64 {
	mask := branchNode.Mask()
	childIds := branchNode.Children()
	childCount := len(childIds)
	var newChildHashes [][]byte
	var newChildIds []uint64
	var newNibbleMask uint16 = 1 << newNodeNibble
	newMask := mask | newNibbleMask
	var offset int

	if newMask == mask {
		// branch has child node with same nibble as newNodeNibble
		newChildIds = make([]uint64, childCount)
		newChildHashes = make([][]byte, childCount)
		offset = 0
	} else {
		// branch does not have any child node with same nibble as newNodeNibble
		newChildIds = make([]uint64, childCount+1)
		newChildHashes = make([][]byte, childCount+1)
		offset = 1
	}

	currentMask := newMask
	for i := 0; i < childCount+offset; i++ {
		// getting lowest nibble-mask
		// removing lowest nibble-mask from next-mask
		nextMask := currentMask & (currentMask - 1)
		nibbleMask := currentMask ^ nextMask

		var currentNodeId uint64
		if nibbleMask < newNibbleMask {
			currentNodeId = childIds[i]
		} else if nibbleMask == newNibbleMask {
			currentNodeId = newNodeId
		} else {
			currentNodeId = childIds[i-offset]
		}

		currentNode, err := trie.GetNodeById(currentNodeId)
		if err != nil {
			panic(err)
		}
		newChildIds[i] = currentNodeId
		newChildHashes[i] = currentNode.Hash()

		currentMask = nextMask
	}

	return trie.newBranchNode(newMask, newChildIds, newChildHashes)
}

func (trie *TrieHashMap) updateTrieExtension(
	extensionNode *nodes.Node, matched byte, nibbleHeight byte, keyHash []byte, value []byte,
) uint64 {
	// we got a new leaf node
	newLeafNodeId := trie.newLeafNode(keyHash, value)
	newLeafNode, err := trie.GetNodeById(newLeafNodeId)
	if err != nil {
		panic(err)
	}

	nibbleCount := extensionNode.NibbleCount()
	nibbles := extensionNode.Nibbles()
	remainingNibble := nibbleCount - matched
	// this child node is a branch node with at least 2 children
	oldNodeChildId := extensionNode.Child()
	oldNodeChild, err := trie.GetNodeById(oldNodeChildId)
	if err != nil {
		panic(err)
	}

	var newBranchNodeId uint64 = 0
	if remainingNibble == 1 {
		// new branch node with 2 children: child of current extension node and new leaf node
		newBranchNodeId = trie.newBranchNodeFromLeaves(
			oldNodeChildId, newLeafNodeId, utils.GetNibble(nibbles, int(nibbleCount-1)),
			utils.GetNibble(keyHash, int(nibbleHeight+matched)), oldNodeChild.Hash(), newLeafNode.Hash(),
		)
	} else if remainingNibble == 2 {
		// intermediate branch node is created with only one child: child of current extension node
		// remember, child of current extension node is also a branch node with at least 2 children
		// so the intermediate branch node is correct
		tempBranchNodeId := trie.newBranchNode(
			1<<utils.GetNibble(nibbles, int(nibbleCount-1)), []uint64{oldNodeChildId},
			[][]byte{oldNodeChild.Hash()},
		)
		tempBranchNode, err := trie.GetNodeById(tempBranchNodeId)
		if err != nil {
			panic(err)
		}
		// new branch node with 2 children: created intermediate branch node above and new leaf node
		newBranchNodeId = trie.newBranchNodeFromLeaves(
			tempBranchNodeId, newLeafNodeId, utils.GetNibble(nibbles, int(nibbleCount-2)),
			utils.GetNibble(keyHash, int(nibbleHeight+matched)), tempBranchNode.Hash(), newLeafNode.Hash(),
		)
	} else {
		// a new extension node is created with one child: child of current extension node
		// so it is a correct extension node
		newNibbles := utils.GetNibbles(nibbles, int(nibbleCount-remainingNibble+1), int(remainingNibble-1))
		newExtensionNodeId := trie.newExtensionNode(remainingNibble-1, newNibbles, oldNodeChildId, oldNodeChild.Hash())
		newExtensionNode, err := trie.GetNodeById(newExtensionNodeId)
		if err != nil {
			panic(err)
		}
		// new branch node with 2 children: created extension node above and new leaf node
		newBranchNodeId = trie.newBranchNodeFromLeaves(
			newExtensionNodeId, newLeafNodeId, utils.GetNibble(nibbles, int(nibbleCount-remainingNibble)),
			utils.GetNibble(keyHash, int(nibbleHeight+matched)), newExtensionNode.Hash(), newLeafNode.Hash(),
		)
	}

	if newBranchNodeId == 0 {
		panic("something is wrong")
	}

	if matched == 0 {
		// extension node is replaced by the new branch node
		return newBranchNodeId
	}

	newBranchNode, err := trie.GetNodeById(newBranchNodeId)
	if err != nil {
		panic(err)
	}
	if matched == 1 {
		// extension node is replaced by another branch node with 1 child: the new branch node created above
		// the new branch node created above has 2 children, so the replacement is correct
		return trie.newBranchNode(
			1<<utils.GetNibble(nibbles, 0), []uint64{newBranchNodeId}, [][]byte{newBranchNode.Hash()},
		)
	} else {
		// more than 1 nibbles matched, so the extension node will remain, but nibbles will be updated
		// its child will be the new branch node created above
		return trie.newExtensionNode(
			matched, utils.GetNibbles(nibbles, 0, int(matched)), newBranchNodeId, newBranchNode.Hash(),
		)
	}
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
		return trie.newBranchNode(1<<nibbles[0], []uint64{newBranchId}, [][]byte{newBranchNode.Hash()})
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

	return trie.newBranchNode(uint16((1<<oldNodeNibble)|(1<<newNodeNibble)), childIds, childHashes)
}

func (trie *TrieHashMap) newBranchNode(mask uint16, childIds []uint64, childHashes [][]byte) uint64 {
	newNode, err := nodes.NewBranchNode(mask, childIds, childHashes)
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
		return trie.nodeRepo.GetNodeById(nodeId)
	} else {
		return node, nil
	}
}
