package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"time"
)

type ReqCache struct {
	db  *sql.DB
	log *log.Logger
}

const dbFile string = "data/cache.db"

const reqTable string = `
  CREATE TABLE IF NOT EXISTS reqdata (
      httpdata BLOB NOT NULL,
      hash TEXT NOT NULL,
      expiry INT NOT NULL
  )
`

func NewReqCache(cfg *Config) *ReqCache {
	logger := log.New(os.Stderr, "(cache)", log.LstdFlags)
	db, err := sql.Open("sqlite3", "file:"+dbFile)
	dbError(logger, err)
	_, err = db.Exec(reqTable)

	dbError(logger, err)
	rc := ReqCache{
		db:  db,
		log: logger,
	}
	go rc.purgeExpired()
	return &rc
}

func (rc *ReqCache) purgeExpired() {
	for {
		expiry := time.Now().Unix()
		_, err := rc.db.Exec("DELETE FROM reqdata WHERE expiry < ?", expiry)
		if err != nil {
			dbError(rc.log, err)
		}
		time.Sleep(1 * time.Hour)
	}
}

func (rc *ReqCache) CachedFetch(req *http.Request, client *http.Client) (*http.Response, error) {
	reqBytes, _ := httputil.DumpRequest(req, true)
	md5Hash := md5.Sum(reqBytes)
	reqHash := hex.EncodeToString(md5Hash[:])
	row := rc.db.QueryRow("SELECT httpdata FROM reqdata WHERE hash = ?", reqHash)
	var httpdata []byte
	if row.Scan(&httpdata) == nil {
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(httpdata)), req)
		return res, err
	}
	//binary.Write(os.Stdout, binary.LittleEndian, reqBytes)
	//os.Exit(2)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	rc.log.Println("MISS", req.URL.Host)
	_, err = rc.db.Exec("INSERT INTO reqdata VALUES (?,?,?)",
		respBytes,
		reqHash,
		time.Now().Unix()+86400,
	)
	if err != nil {
		dbError(rc.log, err)
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(respBytes)), req)
}

func dbError(log *log.Logger, err error) {
	if err != nil {
		log.Println("DB Error", err.Error())
		debug.PrintStack()
		os.Exit(4)
	}
}
