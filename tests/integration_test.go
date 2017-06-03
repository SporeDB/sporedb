package tests

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScenario(t *testing.T) {
	var out string
	path, end, err := setup("testdata")
	defer end()
	require.Nil(t, err)

	clients := []string{"alice", "bob", "carol"}
	ports := []string{"4000", "4100", "4200"}

	// Setup private keys
	for _, c := range clients {
		t.Log("Creating private key for", c)
		out, err = execute(path, c, "keys init", "")
		require.Nil(t, err, out)
	}

	// Export public keys
	pubKeys := make([]string, len(clients))
	for i, c := range clients {
		t.Log("Exporting public key of", c)
		out, err = execute(path, c, "keys export", "")
		require.Nil(t, err, out)
		pubKeys[i] = out
	}

	// Build trust network:
	// * Alice trusts Bob and Carol, and signs Bob
	t.Log("Building trust network")
	out, err = execute(path, "alice", "keys import bob -t high", pubKeys[1])
	assert.Nil(t, err, out)
	out, err = execute(path, "alice", "keys import carol -t high", pubKeys[2])
	assert.Nil(t, err, out)
	out, err = execute(path, "alice", "keys sign bob", "")
	assert.Nil(t, err, out)
	out, err = execute(path, "alice", "keys export", "")
	assert.Nil(t, err, out)
	pubKeys[0] = out

	// * Bob trusts Alice and Carol
	out, err = execute(path, "bob", "keys import alice -t high", pubKeys[0])
	assert.Nil(t, err, out)
	out, err = execute(path, "bob", "keys import carol -t high", pubKeys[2])
	assert.Nil(t, err, out)

	// * Carol trusts Alice and rely on Alice to trust Bob
	out, err = execute(path, "carol", "keys import alice -t high", pubKeys[0])
	assert.Nil(t, err, out)
	out, err = execute(path, "carol", "keys import bob -t none", pubKeys[1])
	assert.Nil(t, err, out)

	// Check trust network
	for _, c := range clients {
		out, _ = execute(path, c, "keys ls", "")
		t.Logf("Stored keys for %s:\n%s", c, out)
	}

	// Create policy file
	t.Log("Creating policy file (by alice)")
	stdin := `policy
⌐■-■
y
bob
carol

2

`

	out, err = execute(path, "alice", "policy create", stdin)
	assert.Nil(t, err, out)
	policyData, err := ioutil.ReadFile(filepath.Join(path, "policy.json"))
	assert.Nil(t, err)
	t.Log(string(policyData))

	// Start the servers!
	t.Log("Running the servers and testing the clients")
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, c := range clients {
		go func(c string) {
			out, _ = execute(path, c, "server", "")
			t.Logf("Server output for %s:\n%s", c, out)
			wg.Done()
		}(c)
		time.Sleep(time.Second)
	}

	// Start the client
	stdin = `SET foo bar
ADD cmp 1
ADD cmp 11
SADD mem fourty
SADD mem two
`
	out, err = execute(path, "bob", "client -p policy -s localhost:"+ports[1], stdin)
	assert.Nil(t, err, out)

	// Wait a bit for propagation
	time.Sleep(5 * time.Second)

	// Test client's values
	stdin = `GET foo
GET cmp
VERSION cmp
SMEMBERS mem
`

	var res string
	for i, c := range clients {
		out, _ = execute(path, c, "client -p policy -s localhost:"+ports[i], stdin)
		t.Logf("Client output for %s:\n%s", c, out)
		if i == 0 {
			res = out
		} else {
			require.Exactly(t, res, out, fmt.Sprintf("result must be equal between %s and %s", clients[0], clients[i]))
		}

		require.Contains(t, out, "fourty")
		require.Contains(t, out, "two")
	}

	wg.Wait()
}
