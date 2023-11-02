package utils

import (
	"encoding/json"
	"errors"
	"github.com/patrickmn/go-cache"
	"net/http"
	"time"
)

const PINS_URL = "http://pinning.solenopsys.org/select/pins"
const LIB_KEY = "front.static.library"
const LIB_VALUE_DEFAULT = "*"

type SharedCache struct {
	librariesMapping map[string]string
	cache            *cache.Cache
	httpLoader       HttpLoader
}

func (sc *SharedCache) LoadMapping(value string) error {
	url := PINS_URL + "?name=" + LIB_KEY + "&value=" + value
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jsonDecoded map[string]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&jsonDecoded); err != nil {
		return err
	}

	for key, value := range jsonDecoded {
		libName := value[LIB_KEY]
		sc.librariesMapping[libName] = key
	}

	return nil
}

func (sc *SharedCache) LoadMappingAll() error {
	return sc.LoadMapping(LIB_VALUE_DEFAULT)
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

func (sc *SharedCache) GetLibRemote(libName string) ([]byte, error) {
	EMPTY := []byte{}
	libCid := sc.librariesMapping[libName]
	if libCid == "" {
		err := sc.LoadMapping(libName)
		if err != nil {
			return EMPTY, err
		}
		libCid = sc.librariesMapping[libName]
		if libCid == "" {
			return EMPTY, errors.New("library not found")
		}
	}
	return sc.httpLoader.httpGet(libCid)
}

func NewSharedCache(ipfsHosts []string, expiration time.Duration, loadThreads int) *SharedCache {
	return &SharedCache{
		librariesMapping: make(map[string]string),
		cache:            cache.New(expiration, expiration*2),
		httpLoader:       *NewHttpLoader(ipfsHosts, loadThreads),
	}
}
