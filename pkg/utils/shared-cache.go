package utils

import (
	"github.com/patrickmn/go-cache"
	"time"
)

const LIB_KEY_DEFAULT = "front.static.library"
const LIB_VALUE_DEFAULT = "*"

type SharedCache struct {
	pinningMapping *PinningMapping
	cache          *cache.Cache
	httpLoader     *HttpLoader
}

func (sc *SharedCache) GetLib(libName string) ([]byte, error) {
	bytes, b := sc.cache.Get(libName)
	if b {
		return bytes.([]byte), nil
	} else {
		bytes, err := sc.GetLibRemote(libName)
		if err != nil {
			return nil, err
		}
		sc.cache.Set(libName, bytes, cache.DefaultExpiration)
		return bytes, nil
	}
}

func (sc *SharedCache) LoadMappingAll() error {
	return sc.pinningMapping.LoadMapping(LIB_VALUE_DEFAULT)
}

func (sc *SharedCache) GetLibRemote(libName string) ([]byte, error) {
	libCid, err := sc.pinningMapping.GetCid(libName)
	if err != nil {
		return nil, err
	}
	return sc.httpLoader.httpGet(libCid, false)
}

func NewSharedCache(httpLoader *HttpLoader, expiration time.Duration) *SharedCache {
	return &SharedCache{
		pinningMapping: NewPinningMapping(LIB_KEY_DEFAULT, LIB_VALUE_DEFAULT, LIB_KEY_DEFAULT, false),
		cache:          cache.New(expiration, expiration*2),
		httpLoader:     httpLoader,
	}
}
