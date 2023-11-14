package utils

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"k8s.io/klog/v2"
)

type Router struct {
	loader     *HttpLoader
	nextUid    uint64
	dagLoaders map[uint64]DagIntf
}

func NewRouter(loader *HttpLoader) *Router {
	loader.Start()
	router := &Router{
		loader:     loader,
		nextUid:    0,
		dagLoaders: make(map[uint64]DagIntf),
	}
	go router.StartRoute()
	return router
}

func (r *Router) StartRoute() {
	for packet := range r.loader.responses {
		if dagLoader, ok := r.dagLoaders[packet.Sender]; ok {
			dagLoader.ResHttp(packet)
		} else {
			klog.Error("Error in dag loader: ", packet.Sender)
		}
	}
}

func (r *Router) LoadNode(cid string, dagLoader DagIntf) datamodel.Node {
	r.nextUid++

	dagLoader.SetUid(r.nextUid)

	dagLoader.SetCidFunc(func(packet *HttpPacket) {
		r.loader.cids <- packet
	})
	r.dagLoaders[r.nextUid] = dagLoader
	defer delete(r.dagLoaders, r.nextUid)
	return dagLoader.Load()
}
