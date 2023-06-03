package nodes

import (
	"errors"
	"merkle-patricia-tree/utils"
)

func NewExtensionNode(nibbleCount byte, nibbles []byte, child uint64, childNodeHash []byte) (*Node, error) {
	if nibbleCount < 2 {
		return nil, errors.New("got less then 2 nibbles for extension type node")
	}
	node := new(Node)
	node.nodeType = Extension
	node.nibblesCountOddParity = (nibbleCount & 1) != 0
	node.nibbles = make([]byte, len(nibbles))
	copy(node.nibbles, nibbles)
	node.children = []uint64 {child}
	hash, err := node.ComputeExtensionNodeHash(childNodeHash)
	if err != nil {
		return nil, err
	}
	node.hash = make([]byte, HashLen)
	copy(node.hash, hash)
	return node, nil
}

func (node *Node) Child() uint64 {
	return node.children[0]
}

func (node *Node) NibbleCount() byte {
	if node.nibblesCountOddParity {
		return byte((len(node.nibbles) << 1) - 1)
	} else {
		return byte(len(node.nibbles) << 1)
	}
}

func (node *Node) Nibbles() []byte {
	nibbles := make([]byte, len(node.nibbles))
	copy(nibbles, node.nibbles)
	return nibbles
}

func (node *Node) ComputeExtensionNodeHash(childNodeHash []byte) ([]byte, error) {
	if node.nodeType != Extension {
		return nil, errors.New("Incorrect node type for ComputeExtensionNodeHash()")
	}
	var hash []byte
	if node.nibblesCountOddParity {
		hash = append(hash, 1)
	} else {
		hash = append(hash, 0)
	}
	hash = append(hash, node.nibbles...)
	hash = append(hash, childNodeHash...)
	hash = utils.Keccak256(hash)
	if len(hash) != HashLen {
		return nil, errors.New("node hash has incorrect length")
	}
	return hash, nil
}

// TODO: implement serializer
func (node *Node) ExtensionNodeToBytes() []byte {
	
}

func ExtensionNodeFromBytes(bytes []byte) (*Node, error) {
	
}