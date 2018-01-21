package p2p

import (
	"testing"
	"os"
	"github.com/libp2p/go-libp2p-peerstore"
)

func TestAkhHost_DiscoverPeers(t *testing.T) {

	os.Remove(HostsInfoPath)

	h := make([]AkhHost, 3, 3)

	for i := 0; i < 3; i++ {
		h[i] = StartHost(9842 + i)
	}

	for i := 0; i < 3; i++ {
		if len(h[i].Peerstore().Peers()) > 1 {
			t.Error("Unexpected N of peers: > 1")
		}
	}

	h0Info := h[0].Peerstore().PeerInfo(h[0].ID())
	h0Info.Addrs = append(h0Info.Addrs, h[0].Addrs()...)

	h[1].testPeer(h0Info)
	h[1].addPeer(h0Info)

	h1Info := h[1].Peerstore().PeerInfo(h[1].ID())
	h1Info.Addrs = append(h1Info.Addrs, h[1].Addrs()...)

	h[2].populatePeerStore([]peerstore.PeerInfo{h1Info})

	for i := 0; i < 3; i++ {
		peersLen := len(h[i].Peerstore().Peers())
		if peersLen != 3 {
			t.Errorf("Peer %d has unexpected N of peers: %d", i, peersLen)
		}
		h[i].Close()
	}
}