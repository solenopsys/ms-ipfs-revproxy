package main

import (
	"bytes"
	"fmt"
	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/patrickmn/go-cache"
	"time"
)

type DagCache struct {
	cache      *cache.Cache
	conf       map[string][]string
	hosts      []string
	expiration time.Duration
	router     *Router
}

func NewDagCache(hosts []string, expiration time.Duration, loadThreads int, conf map[string][]string) *DagCache {

	return &DagCache{
		cache:      cache.New(expiration, expiration*2),
		conf:       conf,
		hosts:      hosts,
		expiration: expiration,
		router:     NewRouter(NewHttpLoader(hosts, loadThreads)),
	}
}

func (dc *DagCache) processQuery(key string, cid string) ([]byte, error) {

	id := key + cid

	if dc.conf[key] != nil {
		//load
		if data, found := dc.cache.Get(id); found {
			return data.([]byte), nil
		} else {
			node := dc.router.LoadNode(cid, dc.conf[key])
			byteWriter := bytes.NewBuffer([]byte{})

			err := json.Encode(node, byteWriter)
			if err != nil {
				return nil, err
			}
			data := byteWriter.Bytes()
			dc.cache.Set(id, data, dc.expiration)

			return data, nil
		}
	} else {
		return nil, fmt.Errorf("type not found")
	}
}
