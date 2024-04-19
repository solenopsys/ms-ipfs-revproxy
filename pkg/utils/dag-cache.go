package utils

import (
	"bytes"
	"time"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/patrickmn/go-cache"
)

type DagCache struct {
	cache      *cache.Cache
	conf       map[string][]string
	expiration time.Duration
	router     *Router
}

func NewDagCache(httpLoader *HttpLoader, expiration time.Duration, conf map[string][]string) *DagCache {

	return &DagCache{
		cache: cache.New(expiration, expiration*2),
		conf:  conf,

		expiration: expiration,
		router:     NewRouter(httpLoader),
	}
}

func (dc *DagCache) ProcessQuery(key string, cid string) ([]byte, error) {

	id := key + cid

	if data, found := dc.cache.Get(id); found {
		return data.([]byte), nil
	} else {

		keys := dc.conf[key]
		var dagLoader DagIntf

		dagLoader = NewRecursiveDagLoader(cid, keys)

		node := dc.router.LoadNode(cid, dagLoader)
		byteWriter := bytes.NewBuffer([]byte{})

		err := json.Encode(node, byteWriter)
		if err != nil {
			return nil, err
		}
		data := byteWriter.Bytes()
		dc.cache.Set(id, data, dc.expiration)

		return data, nil
	}
}
