package nodes

import "errors"

type NodeType byte

const (
	HashLen = 32
)

const (
	Leaf NodeType = 0
	Extension NodeType = 1
	Branch NodeType = 2
)

type Node struct {
	nodeType 	NodeType

	// (key,value) pair for leaf type node
	keyHash 				[]byte
	value 					[]byte

	// child node ids for branch type node
	// only one child for extension type node
	children 				[]uint64
	// mask represent which of all 16 nibbles are to use in the branch
	mask 					uint16

	// nibbles for extension type node
	nibbles 				[]byte
	// true for odd number of nibbles and false otherwise
	nibblesCountOddParity 	bool

	// hash of the node
	hash 					[]byte

}

func (node *Node) Type() NodeType {
	return node.nodeType
}

func (node *Node) Hash() []byte {
	hash := make([]byte, HashLen)
	copy(hash, node.hash)
	return hash
}

func (node *Node) ToBytes() ([]byte, error) {
	if node.nodeType == Leaf {
		return node.LeafNodeToBytes(), nil
	} else if node.nodeType == Branch {
		return node.BranchNodeToBytes(), nil
	} else if node.nodeType == Extension {
		return node.ExtensionNodeToBytes(), nil
	} else {
		return nil, errors.New("Unsupported node type")
	}
}

func FromBytes(bytes []byte) (*Node, error) {
	if len(bytes) < 1 {
		return nil, errors.New("cannot decode node, incorrect byte array")
	}
	if bytes[0] == byte(Leaf) {
		return LeafNodeFromBytes(bytes)
	} else if bytes[0] == byte(Branch) {
		return BranchNodeFromBytes(bytes)
	} else if bytes[0] == byte(Extension) {
		return ExtensionNodeFromBytes(bytes)
	} else {
		return nil, errors.New("cannot decode node, incorrect byte array")
	}
}

// func (node *Node) Test() {
// 	fmt.Println("done")
// }