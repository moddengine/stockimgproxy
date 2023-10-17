package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

type ReqCache struct {
	store *Store
	log   *log.Logger
}

func NewReqCache(cfg *Config, store *Store) *ReqCache {
	logger := log.New(os.Stderr, "(cache) ", log.LstdFlags)
	rc := ReqCache{
		store: store,
		log:   logger,
	}
	go rc.purgeExpired()
	return &rc
}

func (rc *ReqCache) purgeExpired() {
	for {
		expiry := time.Now().Unix()
		rc.store.DeleteBefore(expiry)
		time.Sleep(1 * time.Hour)
	}
}

func (rc *ReqCache) CachedFetch(req *http.Request, client *http.Client) (*http.Response, error) {
	reqBytes, _ := httputil.DumpRequest(req, true)
	md5Hash := md5.Sum(reqBytes)
	reqHash := hex.EncodeToString(md5Hash[:])
	data, ok := rc.store.GetResponse(reqHash)
	if ok {
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), req)
		if err == nil {
			return res, nil
		} else {
			rc.log.Println("Problems decoding cached result", err.Error())
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	rc.log.Println("MISS", req.URL.Host)
	rc.store.StoreResponse(reqHash, respBytes, time.Now().Unix()+86400)
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(respBytes)), req)
}
