package utils

import (
	"io/ioutil"
	"math/rand"
	"net/http"

	"github.com/patrickmn/go-cache"
	"k8s.io/klog/v2"
)

type HttpPacket struct {
	Payload interface{}
	Sender  uint64
}

type HttpResponse struct {
	Cid       string
	Data      []byte
	FromCache bool
	Error     error
}

type HttpLoader struct {
	ipfsHosts []string
	threads   int
	cids      chan *HttpPacket
	responses chan *HttpPacket
	cache     *cache.Cache
}

func NewHttpLoader(ipfsHosts []string, threads int) *HttpLoader {
	return &HttpLoader{
		ipfsHosts: ipfsHosts,
		threads:   threads,
		cids:      make(chan *HttpPacket, 1024),
		responses: make(chan *HttpPacket, 1024),
		cache:     cache.New(cache.NoExpiration, cache.NoExpiration),
	}
}

func (dl *HttpLoader) RandomHost() string {
	id := rand.Intn(len(dl.ipfsHosts))
	return dl.ipfsHosts[id]
}

func (dl *HttpLoader) Start() {
	for i := 0; i < dl.threads; i++ {
		go dl.load()
	}
}

func (dl *HttpLoader) load() {
	for request := range dl.cids {
		sCid := request.Payload.(string)

		if data, found := dl.cache.Get(sCid); found {
			dl.responses <- &HttpPacket{
				Payload: data.(*HttpResponse),
				Sender:  request.Sender,
			}
			continue
		}
		data, err := dl.httpGet(sCid, false)
		if err != nil {
			klog.Error("Error http reques: ", err)
			// dl.wg.Done()
		}
		dl.responses <- &HttpPacket{
			Payload: &HttpResponse{sCid, data, false, err},
			Sender:  request.Sender,
		}

		err = dl.cache.Add(sCid, &HttpResponse{sCid, data, true, err}, cache.NoExpiration)
		if err != nil {
			klog.Error("Cache add error: ", err)
		}
	}
}

func (dl *HttpLoader) httpGetSubFile(cid string, filePath string, byName bool, json bool) ([]byte, error) {
	var typeSelect = "ipfs"
	if byName {
		typeSelect = "ipns"
	}

	var url = "http://" + dl.RandomHost() + "/" + typeSelect + "/" + cid + filePath
	if json {
		url = url + "?format=dag-json"
	}
	klog.Info("Load from: ", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	all, err := ioutil.ReadAll(resp.Body)
	return all, err
}

func (dl *HttpLoader) httpGet(cid string, byName bool) ([]byte, error) {
	return dl.httpGetSubFile(cid, "", byName, false)
}

func (dl *HttpLoader) httpGetJson(cid string, byName bool) ([]byte, error) {
	return dl.httpGetSubFile(cid, "", byName, true)
}
