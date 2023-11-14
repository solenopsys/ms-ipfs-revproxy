package utils

import "github.com/ipld/go-ipld-prime/datamodel"

type DagIntf interface {
	SetUid(uid uint64)
	Load() datamodel.Node
	ResHttp(resp *HttpPacket)
	SetCidFunc(fun func(packet *HttpPacket))
}
