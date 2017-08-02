package myc

import (
	"testing"

	"gitlab.com/SporeDB/sporedb/myc/protocol"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/require"
)

func Test_requestsContainer(t *testing.T) {
	cache, _ := lru.New(5)
	rc := &requestsContainer{
		cache:       cache,
		maxRequests: 2,
	}

	n := 10

	output := make(chan string, n)
	nodes := []protocol.Node{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
		{Address: "d"},
	}

	for i := 0; i < n; i++ {
		go func(i int) {
			node := nodes[i%len(nodes)]
			transmit := rc.Add("1", node)
			if transmit {
				output <- node.Address
			} else {
				output <- ""
			}
		}(i)
	}

	var transmitted []string

	for i := 0; i < n; i++ {
		o := <-output
		if o != "" {
			transmitted = append(transmitted, o)
		}
	}

	require.True(t, len(transmitted) >= 2, "should ask for at least 2 requests")
	require.True(t, existDifferent(transmitted), "should ask to 2 different nodes")

	rc.SetDelivered("2", []byte{1})
	require.False(t, rc.Add("2", nodes[0]), "should not request a delivered spore")
}

func existDifferent(s []string) bool {
	if len(s) < 2 {
		return false
	}

	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return true
		}
	}

	return false
}
