package utils

import (
	"context"
	"strings"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"k8s.io/klog/v2"
)

// todo need deep refactoring it trash
type RecursiveDagLoader struct { // todo remove it
	rootCid       string
	recursiveKeys []string
	nodes         map[string]*datamodel.Node
	raws          map[string][]byte
	wg            *sync.WaitGroup
	uid           uint64
	ctx           context.Context
	cancel        context.CancelFunc
	sendCidFunc   func(packet *HttpPacket)
}

func (dl *RecursiveDagLoader) SetCidFunc(fun func(packet *HttpPacket)) {
	dl.sendCidFunc = fun
}

func NewRecursiveDagLoader(rootCid string, recursiveKeys []string) *RecursiveDagLoader {
	ctx, cancel := context.WithCancel(context.Background())

	return &RecursiveDagLoader{
		rootCid:       rootCid,
		recursiveKeys: recursiveKeys,
		nodes:         make(map[string]*datamodel.Node),
		raws:          make(map[string][]byte),
		wg:            &sync.WaitGroup{},
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (dl *RecursiveDagLoader) SetUid(uid uint64) {
	dl.uid = uid

}

func (dl *RecursiveDagLoader) keyContain(key string) bool {
	for _, k := range dl.recursiveKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (dl *RecursiveDagLoader) nodeFromCache(cidString string) datamodel.Node {
	cidObj, err := cid.Decode(cidString)
	if err != nil {
		klog.Error("Error in cid decode: ", err)
	}

	if cidObj.Type() == cid.DagProtobuf {
		np := basicnode.Prototype.Any // Pick a stle for the in-memory data.
		builder := np.NewBuilder()
		builder.AssignString(string(dl.raws[cidString]))
		return builder.Build()

	} else {
		cachedNode := dl.nodes[cidString]
		return dl.ScanNode(*cachedNode, true, cidString)
	}

}

func (dl *RecursiveDagLoader) ScanNode(node datamodel.Node, transform bool, cid string) datamodel.Node {
	iterator := node.MapIterator()

	np := basicnode.Prototype.Any // Pick a stle for the in-memory data.
	builder := np.NewBuilder()
	beginMap, err := builder.BeginMap(node.Length())
	if err != nil {
		klog.Fatal(err)
	}
	if cid != "" {
		beginMap.AssembleKey().AssignString("cid")
		beginMap.AssembleValue().AssignString(cid)
	}

	if iterator != nil {
		for !iterator.Done() {
			key, value, err := iterator.Next()
			if err != nil {
				panic(err)
			}

			keyString, err := key.AsString()

			inject := dl.keyContain(keyString)
			//if inject  {

			if err != nil {
				klog.Fatal(err)
			}
			l := value.Length()

			if l >= 0 {
				bl := dl.transformList(value, transform, inject)
				if transform {
					beginMap.AssembleKey().AssignNode(key)
					beginMap.AssembleValue().AssignNode(bl)
				}
			} else {
				link, err := value.AsLink()

				if err != nil {
					if transform {
						beginMap.AssembleKey().AssignNode(key)
						beginMap.AssembleValue().AssignNode(value)
					}
				} else {
					cidLink := link.String()
					if transform {
						if cid == "" {
							beginMap.AssembleKey().AssignString("cid")
							beginMap.AssembleValue().AssignString(cidLink)
						}

						beginMap.AssembleKey().AssignNode(key)
						//	beginMap.AssembleValue().AssignNode(dl.nodeFromCache(cidLink))

						if inject {
							beginMap.AssembleValue().AssignNode(dl.nodeFromCache(cidLink))
						} else {
							beginMap.AssembleValue().AssignString(link.String())
						}
					} else {
						dl.wg.Add(1)
						dl.sendCidFunc(&HttpPacket{Payload: cidLink, Sender: dl.uid})
					}
				}

			}
		}
		beginMap.Finish()
		return builder.Build()
	}

	return node
}

func (dl *RecursiveDagLoader) transformList(sourceNode datamodel.Node, transform bool, inject bool) datamodel.Node {

	np := basicnode.Prototype.Any
	builderList := np.NewBuilder()
	transformedNode, err := builderList.BeginList(sourceNode.Length())
	if err != nil {
		klog.Fatal(err)
	}

	listIterator := sourceNode.ListIterator()

	if listIterator != nil {
		for !listIterator.Done() {
			_, v, err := listIterator.Next()
			if err != nil {
				klog.Fatal(err)
			}

			link, err := v.AsLink()
			if err != nil {
				subnode := dl.ScanNode(v, transform, "")
				transformedNode.AssembleValue().AssignNode(subnode)
			} else {
				cidLoad := link.String()

				if transform {
					if inject {
						transformedNode.AssembleValue().AssignNode(dl.nodeFromCache(cidLoad))
					} else {
						transformedNode.AssembleValue().AssignString(link.String())
					}
				} else if inject {
					dl.wg.Add(1)
					dl.sendCidFunc(&HttpPacket{Payload: cidLoad, Sender: dl.uid})
				}
			}

		}
	} else if sourceNode.MapIterator() != nil {
		return sourceNode
	}

	transformedNode.Finish()
	return builderList.Build()

}

func (dl *RecursiveDagLoader) ProcessResponse(resp *HttpResponse) {
	if resp.Error != nil {
		klog.Error("Error in http response: ", resp.Error)
	}

	c, err := cid.Decode(resp.Cid)
	if err != nil {
		klog.Error("Error in cid decode: ", err)
	}

	if c.Type() == cid.DagCBOR {
		dataReader := strings.NewReader(string(resp.Data))
		np := basicnode.Prototype.Any
		builder := np.NewBuilder()
		dagcbor.Decode(builder, dataReader)
		node := builder.Build()

		dl.nodes[resp.Cid] = &node

		println("Cid: ", resp.Cid, " ", node)
		dl.ScanNode(node, false, resp.Cid)
	}

	if c.Type() == cid.DagProtobuf {

		// Decode the protobuf message into a MyMessage struct

		dl.raws[resp.Cid] = resp.Data
	}

}

func (dl *RecursiveDagLoader) ResHttp(resp *HttpPacket) {
	response := resp.Payload.(*HttpResponse)
	if resp.Sender == dl.uid {
		dl.ProcessResponse(response)

		if response.Error != nil {
			klog.Error("WG DONE ERR: ", response.Cid, response.Error)
		}
		dl.wg.Done()
	}
}

func (dl *RecursiveDagLoader) Load() datamodel.Node {
	klog.Info("Loading dag for cid: ", dl.rootCid, " ", dl.uid)
	dl.wg.Add(1)

	dl.sendCidFunc(&HttpPacket{Payload: dl.rootCid, Sender: dl.uid})

	dl.wg.Wait()

	klog.Info("Dag loaded")

	node := dl.ScanNode(*dl.nodes[dl.rootCid], true, dl.rootCid)

	dl.cancel()
	return node
}
