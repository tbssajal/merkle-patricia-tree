package storage

import "github.com/linxGnu/grocksdb"

type RocksDB struct {
	dataBase		*grocksdb.DB
	readOptions		*grocksdb.ReadOptions
	writeOptions	*grocksdb.WriteOptions
}

func NewDeafaultDB() *RocksDB {
	return NewDBWithPath("./db")
}

func NewDBWithPath(dbPath string) *RocksDB {
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(grocksdb.NewLRUCache(3 << 30))

	opts := grocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		panic(err)
	}
	ro := grocksdb.NewDefaultReadOptions()
	wo := grocksdb.NewDefaultWriteOptions()

	rocksDB := new(RocksDB)
	rocksDB.dataBase = db
	rocksDB.readOptions = ro
	rocksDB.writeOptions = wo

	return rocksDB
}

func (rocksDb *RocksDB) Get(key []byte) []byte {
	value, err := rocksDb.dataBase.Get(rocksDb.readOptions, key)
	if err != nil {
		panic(err)
	}
	return value.Data()
}

func (rocksDB *RocksDB) Save(key []byte, value []byte) {
	err := rocksDB.dataBase.Put(rocksDB.writeOptions, key, value)
	if err != nil {
		panic(err)
	}
}

func (rocksDB *RocksDB) Delete(key []byte) {
	err := rocksDB.dataBase.Delete(rocksDB.writeOptions, key)
	if err != nil {
		panic(err)
	}
}

func (rocksDB *RocksDB) SaveBatch(batch *grocksdb.WriteBatch) {
	err := rocksDB.dataBase.Write(rocksDB.writeOptions, batch)
	if err != nil {
		panic(err)
	}
}