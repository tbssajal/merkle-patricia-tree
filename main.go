package main

import (
	"bytes"
	"fmt"
	"merkle-patricia-tree/storage"
	"merkle-patricia-tree/storage/version_handler"
	"merkle-patricia-tree/trie"
	"merkle-patricia-tree/trie/repository"
)

func main() {
	db := storage.NewDeafaultDB()
	versionHandler := version_handler.NewVersionHandler(db)
	nodeRepo := repository.NewNodeRepository(db)
	trie := trie.NewTrieHashMap(versionHandler, nodeRepo)

	var key []byte
	var value []byte
	
	trie.Put(key, value)
	newValue, err := trie.Get(key)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	if !bytes.Equal(value, newValue) {
		msg := "Value not matched"
		fmt.Println(msg)
		panic(msg)
	}

	proof, err := trie.Proof(key)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	if !trie.VerifyProof(key, proof) {
		msg := "proof not verified"
		fmt.Println(msg)
		panic(msg)
	}

	fmt.Println("Done!")

}