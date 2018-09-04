package pool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/vipnode/vipnode/internal/keygen"
	"github.com/vipnode/vipnode/jsonrpc2"
	"github.com/vipnode/vipnode/pool/store"
)

func TestRemotePoolClient(t *testing.T) {
	pool := New()
	pool.skipWhitelist = true

	server, client := jsonrpc2.ServePipe()
	server.Server.Register("vipnode_", pool)

	privkey := keygen.HardcodedKey(t)
	remote := Remote(client, privkey)

	// Add self to pool first, then let's see if we're advised to connect to
	// self (this probably should error at some point but good test for now).
	if err := pool.Store.SetNode(store.Node{ID: "foo", URI: "enode://foo", IsHost: true, Kind: "geth", LastSeen: time.Now()}, ""); err != nil {
		t.Fatal("failed to add host node:", err)
	}
	if err := pool.Store.SetNode(store.Node{ID: "bar", URI: "enode://bar", IsHost: true, Kind: "parity", LastSeen: time.Now()}, ""); err != nil {
		t.Fatal("failed to add host node:", err)
	}

	// This peer will be ignored because LastSeen was too long ago
	if err := pool.Store.SetNode(store.Node{ID: "oldpeer", URI: "enode://oldpeer", IsHost: true, Kind: "parity", LastSeen: time.Now().Add(-5 * store.KeepaliveInterval)}, ""); err != nil {
		t.Fatal("failed to add host node:", err)
	}

	nodes := pool.Store.ActiveHosts("", 3)
	if len(nodes) != 2 {
		t.Errorf("GetHostNodes returned unexpected number of nodes: %d", len(nodes))
	}

	hosts, err := remote.Connect(context.Background(), "geth")
	if err != nil {
		t.Error(err)
	}
	if len(hosts) != 1 {
		t.Fatalf("wrong number of hosts: %d", len(hosts))
	}

	if hosts[0].URI != "enode://foo" {
		t.Errorf("invalid hosts result: %v", hosts)
	}
}

func TestRemotePoolHost(t *testing.T) {
	pool := New()
	pool.skipWhitelist = true

	server, host := jsonrpc2.ServePipe()
	server.Server.Register("vipnode_", pool)

	privkey := keygen.HardcodedKeyIdx(t, 0)
	nodeID := discv5.PubkeyID(&privkey.PublicKey).String()
	remote := Remote(host, privkey)
	kind := "geth"
	payout := ""
	nodeURI := fmt.Sprintf("enode://%s@127.0.0.1:30303", nodeID)

	err := remote.Host(context.Background(), kind, payout, nodeURI)
	if err != nil {
		t.Error(err)
	}

	server2Client, client2Server := jsonrpc2.ServePipe()
	server2Client.Server.Register("vipnode_", pool)

	clientPrivkey := keygen.HardcodedKeyIdx(t, 1)
	remoteClient := Remote(client2Server, clientPrivkey)

	hosts, err := remoteClient.Connect(context.Background(), "geth")
	if err != nil {
		t.Error(err)
	} else if len(hosts) != 1 {
		t.Fatalf("wrong number of hosts: %d", len(hosts))
	}

	resp, err := remote.Update(context.Background(), []string{})
	if err != nil {
		t.Error(err)
	} else if len(resp.InvalidPeers) != 0 {
		t.Errorf("unexpected invalid peers: %s", resp.InvalidPeers)
	}

	hosts, err = remoteClient.Connect(context.Background(), "geth")
	if err != nil {
		t.Error(err)
	} else if len(hosts) != 1 {
		t.Fatalf("wrong number of hosts: %d", len(hosts))
	}
}
