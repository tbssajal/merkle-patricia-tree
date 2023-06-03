package main

import (
	"fmt"
	"merkle-patricia-tree/trie/nodes"
	"merkle-patricia-tree/utils"
)

func main() {
	fmt.Println("main")
	utils.Test()
	var keyHash []byte
	for i := 0; i < 32; i++ {
		keyHash = append(keyHash, byte(i))
	}
	fmt.Println(len(keyHash))
	var value []byte
	node, _ := nodes.NewLeafNode(keyHash, value)
	hash := node.Hash()
	L := len(hash)
	fmt.Println(L)
	fmt.Println(hash)
	hash[0] = 5
	fmt.Println(hash)
	newHash := node.Hash()
	fmt.Println(newHash)
	// node.Ke
}