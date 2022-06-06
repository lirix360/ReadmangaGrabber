package db

import (
	bolt "go.etcd.io/bbolt"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/logger"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

var DBconn *bolt.DB

func init() {
	var err error

	dbFileName := "grabber_data.db"

	checkDBFile := tools.IsFileExist(dbFileName)

	DBconn, err = bolt.Open(dbFileName, 0664, nil)
	if err != nil {
		logger.Log.Fatal("Ошибка при открытии файла БД:", err)
	}

	err = DBconn.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("AppSettings"))
		return err
	})
	if err != nil {
		logger.Log.Fatal("Ошибка при создании бакета (AppSettings) в БД:", err)
	}

	err = DBconn.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("MangaFavs"))
		return err
	})
	if err != nil {
		logger.Log.Fatal("Ошибка при создании бакета (MangaFavs) в БД:", err)
	}

	if !checkDBFile {
		err = DBconn.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("AppSettings"))
			err := b.Put([]byte("dbver"), []byte(config.DBver))
			return err
		})
		if err != nil {
			logger.Log.Fatal("Ошибка при создании стартовой БД:", err)
		}
	}
}
