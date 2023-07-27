package repository

import (
	"errors"
	"merkle-patricia-tree/storage"
	"merkle-patricia-tree/trie/nodes"
	"merkle-patricia-tree/utils"
)

type NodeRepository struct {
	rocksDb 	*storage.RocksDB
}

func NewNodeRepository(rocksDB *storage.RocksDB) *NodeRepository {
	nodeRepo := new(NodeRepository)
	nodeRepo.rocksDb = rocksDB
	return nodeRepo
}

func (nodeRepo *NodeRepository) GetNodeById(nodeId uint64) (*nodes.Node, error) {
	prefix := storage.PrefixNodeById(nodeId)
	value := nodeRepo.rocksDb.Get(prefix)
	if value == nil {
		return nil, errors.New("Node not found in database")
	} else {
		return nodes.FromBytes(value)
	}
}

func (nodeRepo *NodeRepository) GetNodeByHash(hash []byte) (*nodes.Node, error) {
	prefix := storage.PrefixNodeIdByHash(hash)
	value := nodeRepo.rocksDb.Get(prefix)
	if value == nil {
		return nil, errors.New("Node not found in database")
	} else {
		prefix = storage.PrefixNodeById(utils.UInt64FromBytes(value))
		value = nodeRepo.rocksDb.Get(prefix)
		return nodes.FromBytes(value)
	}
}

func (nodeRepo *NodeRepository) SaveNode(nodeId uint64, node *nodes.Node) {
	batch := storage.NewAtomicWrite(nodeRepo.rocksDb)
	// save node by id
	prefix := storage.PrefixNodeById(nodeId)
	bytes, err := node.ToBytes()
	if err != nil {
		panic(err)
	}
	batch.Save(prefix, bytes)

	prefix = storage.PrefixNodeIdByHash(node.Hash())
	batch.Save(prefix, utils.UInt64ToBytes(nodeId))
	batch.Commit()
}