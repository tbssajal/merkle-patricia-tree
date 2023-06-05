package storage

import (
	"sync"

	"github.com/linxGnu/grocksdb"
)

type AtomicWrite struct {
	writeBatch		*grocksdb.WriteBatch
	rocksDb			*RocksDB
	committed		bool
	mutex			*sync.Mutex
}

func NewAtomicWrite(rocksDb *RocksDB) *AtomicWrite {
	atomicWrite := new(AtomicWrite)
	atomicWrite.rocksDb = rocksDb
	atomicWrite.writeBatch = grocksdb.NewWriteBatch()
	atomicWrite.committed = false
	atomicWrite.mutex = &sync.Mutex{}
	return atomicWrite
}

func (atomicWrite *AtomicWrite) Save(key []byte, value []byte) {
	atomicWrite.writeBatch.Put(key, value)
}

func (atomicWrite *AtomicWrite) Delete(key []byte) {
	atomicWrite.writeBatch.Delete(key)
}

func (atomicWrite *AtomicWrite) Commit() {
	atomicWrite.lock()
	defer atomicWrite.unlock()
	
	if atomicWrite.committed {
		panic("AtomicWrite already commited")
	}
	atomicWrite.rocksDb.SaveBatch(atomicWrite.writeBatch)
	atomicWrite.committed = true
}

func (atomicWrite *AtomicWrite) lock() {
	atomicWrite.mutex.Lock()
}

func (atomicWrite *AtomicWrite) unlock() {
	atomicWrite.mutex.Unlock()
}