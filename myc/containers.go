package myc

import (
	lru "github.com/hashicorp/golang-lru"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

type requestsContainer struct {
	cache       *lru.Cache
	maxRequests int
}

type request struct {
	requestedTo []protocol.Node
	delivered   []byte
}

func (rc *requestsContainer) Add(sporeUUID string, node protocol.Node) (transmit bool) {
	// Please note that a race condition might apply here.
	// This is not a concern, as strict request delivery is not required:
	// - In general, no more than n requests should be made for a given Spore
	// - In general, no more than 1 request should be made for a given (Spore, Node) couple
	// - No request should be made for a delivered request
	//
	// BUT those rules are only used for bandwidth optimization and might be bypassed.
	// Also, the cache may be completelly filled and the request lost, that does not really matter.
	//
	// Additionnaly, requests are not stored using pointers to avoid native race write conditions.
	req := request{}

	r, found := rc.cache.Get(sporeUUID)
	if found {
		req = r.(request)

		if req.delivered != nil || len(req.requestedTo) >= rc.maxRequests {
			return
		}

		for _, n := range req.requestedTo {
			if n.Equals(node) {
				return
			}
		}
	}

	req.requestedTo = append(req.requestedTo, node)
	rc.cache.Add(sporeUUID, req)
	transmit = true
	return
}

func (rc *requestsContainer) SetDelivered(sporeUUID string, data []byte) {
	rc.cache.Add(sporeUUID, request{delivered: data})
}

func (rc *requestsContainer) IsDelivered(sporeUUID string) (bool, []byte) {
	r, found := rc.cache.Get(sporeUUID)
	if !found {
		return false, nil
	}

	data := r.(request).delivered
	return data != nil, data
}
