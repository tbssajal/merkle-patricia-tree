package nodes

import (
	"errors"
	"merkle-patricia-tree/utils"
)

const (
	MaxNibbleCount = 16
)

func newBranchNodeGivenHash(mask uint16, children []uint64, hash []byte) *Node {
	node := new(Node)
	node.nodeType = Branch
	node.mask = mask
	node.children = make([]uint64, len(children))
	copy(node.children, children)
	node.hash = make([]byte, HashLen)
	copy(node.hash, hash)
	return node
}

func NewBranchNode(mask uint16, children []uint64, childNodeHashes [][]byte) (*Node, error) {
	if mask == 0 {
		return nil, errors.New("No children of branch type node")
	}
	if len(childNodeHashes) != len(children) || len(GetNibblesFromMask(mask)) != len(children) {
		return nil, errors.New("Missing children")
	}

	node := new(Node)
	node.nodeType = Branch
	node.mask = mask
	node.children = make([]uint64, len(children))
	copy(node.children, children)
	hash, err := ComputeBranchNodeHash(node.mask, childNodeHashes)
	if err != nil {
		return nil, err
	}
	node.hash = make([]byte, HashLen)
	copy(node.hash, hash)
	return node, nil
}

func (node *Node) Mask() uint16 {
	return node.mask
}

func (node *Node) GetChildByNibble(nibble byte) uint64 {
	if nibble >= MaxNibbleCount || (node.mask & (1<<nibble)) == 0 {
		return 0
	} else {
		return node.children[utils.PositionOf(node.mask, nibble)]
	}
}

func (node *Node) ChildrenCount() byte {
	return byte(len(node.children))
}

func (node *Node) Children() []uint64 {
	children := make([]uint64, len(node.children))
	copy(children, node.children)
	return children
}

func ComputeBranchNodeHash(mask uint16, childNodeHashes [][]byte) ([]byte, error) {

	labels := GetNibblesFromMask(mask)
	if len(labels) != len(childNodeHashes) {
		return nil, errors.New("Missing labels or children hash")
	}

	var hash []byte
	childCount := len(labels)
	for i := 0; i < childCount; i++ {
		if len(childNodeHashes[i]) != HashLen {
			return nil, errors.New("Incorrect hash of child node")
		}
		hash = append(hash, labels[i])
		hash = append(hash, childNodeHashes[i]...)
	}

	hash = utils.Keccak256(hash)
	if len(hash) != HashLen {
		return nil, errors.New("node hash has incorrect length")
	}
	return hash, nil
}

func GetNibblesFromMask(mask uint16) []byte {
	var labels []byte
	for i := 0; i < MaxNibbleCount; i++ {
		if (mask & 1) != 0 {
			labels = append(labels, byte(i))
		}
		mask = mask >> 1
	}
	return labels
}

func (node *Node) GetChildNibbles() []byte {
	return GetNibblesFromMask(node.mask)
}

func (node *Node) BranchNodeToBytes() []byte {
	childCount := len(node.children)
	childIdLen := 8
	
	nodeTypeLen := 1
	hashLen := HashLen
	maskLen := 2
	childLen := childCount * childIdLen

	bytes := make([]byte, nodeTypeLen + hashLen + maskLen + childLen)
	pos := 0
	bytes[pos] = byte(node.nodeType)
	pos += nodeTypeLen

	copy(bytes[pos:pos+hashLen], node.hash)
	pos += hashLen

	copy(bytes[pos:pos+maskLen], utils.UInt16ToBytes(node.mask))
	pos += maskLen

	for i := 0 ; i < childCount ; i++ {
		copy(bytes[pos:pos+childIdLen], utils.UInt64ToBytes(node.children[i]))
		pos += childIdLen
	}

	return bytes
}

func BranchNodeFromBytes(bytes []byte) (*Node, error) {
	childCount := 2
	childIdLen := 8
	
	nodeTypeLen := 1
	hashLen := HashLen
	maskLen := 2
	childLen := childCount * childIdLen

	if len(bytes) < nodeTypeLen + hashLen + maskLen + childLen {
		return nil, errors.New("Not enough bytes to decode branch node")
	}

	pos := 0
	if bytes[pos] != byte(Branch) {
		return nil, errors.New("Cannot decode branch node, incorrect node type")
	}
	pos += nodeTypeLen

	hash := make([]byte, hashLen)
	copy(hash, bytes[pos:pos+hashLen])
	pos += hashLen

	mask := utils.UInt16FromBytes(bytes[pos:pos+maskLen])
	pos += maskLen

	childCount = len(GetNibblesFromMask(mask))
	childLen = childCount * childIdLen

	if len(bytes) < nodeTypeLen + hashLen + maskLen + childLen {
		return nil, errors.New("Not enough bytes to decode branch node")
	}

	children := make([]uint64, childCount)
	for i := 0 ; i < childCount ; i++ {
		children[i] = utils.UInt64FromBytes(bytes[pos:pos+childIdLen])
		pos += childIdLen
	}

	return newBranchNodeGivenHash(mask, children, hash), nil
}