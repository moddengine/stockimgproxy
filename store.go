package main

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"github.com/alexedwards/argon2id"
	"github.com/apibillme/cache"
	"log"
	"os"
	"time"
)

type Store struct {
	db        *sql.DB
	log       *log.Logger
	userCache cache.Cache
}

const reqTable string = `
  CREATE TABLE IF NOT EXISTS reqdata (
      httpdata BLOB NOT NULL,
      hash TEXT NOT NULL,
      expiry INT NOT NULL
  )
`

const userTable string = `
  CREATE TABLE IF NOT EXISTS users (
      user TEXT NOT NULL,
      hash TEXT NOT NULL,
      level INT NOT NULL
  )
`

const dbFile string = "data/cache.db"

func NewStore(cfg *Config) *Store {
	logger := log.New(os.Stderr, "(store) ", log.LstdFlags)

	filename := dbFile
	if cfg.Database != "" {
		filename = cfg.Database
	}
	db, err := sql.Open("sqlite3", "file:"+filename)
	dbError(logger, err)

	_, err = db.Exec(reqTable)
	dbError(logger, err)

	_, err = db.Exec(userTable)
	dbError(logger, err)

	userCache := cache.New(256, cache.WithTTL(1*time.Hour))

	return &Store{
		db:        db,
		log:       logger,
		userCache: userCache,
	}
}

func (store *Store) DeleteBefore(expiry int64) {
	_, err := store.db.Exec("DELETE FROM reqdata WHERE expiry < ?", expiry)
	if err != nil {
		dbError(store.log, err)
	}
}

func (store *Store) GetResponse(hash string) ([]byte, bool) {
	row := store.db.QueryRow("SELECT httpdata FROM reqdata WHERE hash = ?", hash)
	var data []byte
	err := row.Scan(&data)
	if err == nil {
		return data, true
	} else {
		println("db:", err.Error())
	}
	return nil, false
}

func (store *Store) StoreResponse(hash string, res []byte, expiry int64) {
	_, err := store.db.Exec("INSERT INTO reqdata VALUES (?,?,?)",
		res,
		hash,
		expiry,
	)
	if err != nil {
		dbError(store.log, err)
	}
}

func (store *Store) TestUser(user string, pass string) bool {
	userPass, ok := store.userCache.Get(user)
	if ok && 1 == subtle.ConstantTimeCompare([]byte(userPass.(string)), []byte(pass)) {
		return true
	}
	row := store.db.QueryRow("SELECT hash FROM users WHERE user = ?", user)
	//println("User:", user, pass)
	var hash string
	err := row.Scan(&hash)
	if err == nil {
		//println("Hash: ", hash)
		match, err := argon2id.ComparePasswordAndHash(pass, hash)
		if err != nil {
			store.log.Println("Error comparing password hashes", err.Error())
			return false
		}
		if match {
			store.userCache.Set(user, pass)
			return true
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		store.log.Println(err.Error())
	}
	return false
}

func dbError(log *log.Logger, err error) {
	if err != nil {
		log.Panicln("DB Error", err.Error())
	}
}
