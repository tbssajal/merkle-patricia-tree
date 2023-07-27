package nodes

import (
	"errors"
	"merkle-patricia-tree/utils"
)

func newExtensionNodeWithHash(nibbleParity bool, nibbles []byte, child uint64, hash []byte) *Node {
	node := new(Node)
	node.nodeType = Extension
	node.nibblesCountOddParity = nibbleParity
	node.nibbles = make([]byte, len(nibbles))
	copy(node.nibbles, nibbles)
	node.children = []uint64 {child}
	node.hash = make([]byte, HashLen)
	copy(node.hash, hash)
	return node
}

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
	hash, err := ComputeExtensionNodeHash(node.nibblesCountOddParity, node.Nibbles(), childNodeHash)
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

func ComputeExtensionNodeHash(nibblesCountOddParity bool, nibbles []byte, childNodeHash []byte) ([]byte, error) {
	var hash []byte
	if nibblesCountOddParity {
		hash = append(hash, 1)
	} else {
		hash = append(hash, 0)
	}
	hash = append(hash, nibbles...)
	hash = append(hash, childNodeHash...)
	hash = utils.Keccak256(hash)
	if len(hash) != HashLen {
		return nil, errors.New("node hash has incorrect length")
	}
	return hash, nil
}

func (node *Node) ExtensionNodeToBytes() []byte {
	nodeTypeLen := 1
	hashLen := HashLen
	nibbleParityLen := 1
	nibbleLen := len(node.nibbles)
	childLen := 8

	bytes := make([]byte, nodeTypeLen + hashLen + nibbleParityLen + nibbleLen + childLen)
	pos := 0
	bytes[pos] = byte(node.nodeType)
	pos += nodeTypeLen

	copy(bytes[pos:pos+hashLen], node.hash)
	pos += hashLen

	if node.nibblesCountOddParity {
		bytes[pos] = 1
	} else {
		bytes[pos] = 0
	}
	pos += nibbleParityLen

	copy(bytes[pos:pos+nibbleLen], node.nibbles)
	pos += nibbleLen

	copy(bytes[pos:pos+childLen], utils.UInt64ToBytes(node.Child()))
	pos += childLen

	return bytes
}

func ExtensionNodeFromBytes(bytes []byte) (*Node, error) {
	nodeTypeLen := 1
	hashLen := HashLen
	nibbleParityLen := 1
	nibbleLen := 1
	childLen := 8

	bytesCount := len(bytes)
	if (bytesCount < nodeTypeLen + hashLen + nibbleParityLen + nibbleLen + childLen) {
		return nil, errors.New("Not enough bytes to decode extension node")
	}

	pos := 0;
	if bytes[pos] != byte(Extension) {
		return nil, errors.New("Cannot decode extension node, incorrect node type")
	}
	pos += nodeTypeLen

	hash := make([]byte, hashLen)
	copy(hash, bytes[pos:pos+hashLen])
	pos += hashLen

	nibbleParity := bytes[pos]
	pos += nibbleParityLen

	nibbleLen = bytesCount - childLen - pos
	nibbles := make([]byte, nibbleLen)
	copy(nibbles, bytes[pos:pos+nibbleLen])
	pos += nibbleLen

	childId := utils.UInt64FromBytes(bytes[pos:pos+childLen])
	return newExtensionNodeWithHash(nibbleParity == 1, nibbles, childId, hash), nil
}