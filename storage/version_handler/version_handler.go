package version_handler

import (
	"merkle-patricia-tree/storage"
	"merkle-patricia-tree/utils"
	"sync"
)

type VersionHandler struct {
	lastVersion uint64
	rocksDb     *storage.RocksDB
	mutex		*sync.Mutex
}


func NewVersionHandler(rocksDB *storage.RocksDB) *VersionHandler {
	versionHandler := new(VersionHandler)
	versionHandler.mutex = &sync.Mutex{}
	versionHandler.rocksDb = rocksDB
	raw := rocksDB.Get(storage.BuildPrefix(storage.LastVersion, []byte {}))
	if raw == nil {
		versionHandler.lastVersion = 0
	} else {
		versionHandler.lastVersion = utils.UInt64FromBytes(raw)
	}
	return versionHandler
}

// TODO implement
func (versionHanlder *VersionHandler) NewVersion() uint64 {
	versionHanlder.lock()
	defer versionHanlder.unlock()
	versionHanlder.lastVersion++
	versionHanlder.rocksDb.Save(
		storage.BuildPrefix(storage.LastVersion, []byte {}), utils.UInt64ToBytes(versionHanlder.lastVersion),
	)
	return versionHanlder.lastVersion
}

func (versionHandler *VersionHandler) lock() {
	versionHandler.mutex.Lock()
}

func (versionHandler *VersionHandler) unlock() {
	versionHandler.mutex.Unlock()
}