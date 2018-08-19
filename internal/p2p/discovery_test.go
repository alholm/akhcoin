package p2p

import (
	"github.com/alholm/akhcoin/pkg/blockchain"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peerstore"
	"os"
	"testing"
)

func init() {
	logging.SetLogLevel("p2p", "DEBUG")
	logging.SetLogLevel("mdns", "DEBUG")
}

func TestAkhHost_DiscoverPeers(t *testing.T) {

	os.Remove(HostsInfoPath)

	var h [3]AkhHost

	for i := 0; i < 3; i++ {
		h[i] = startRandomHost(9765 + i)
	}

	for i := 0; i < 3; i++ {
		if len(h[i].Peerstore().Peers()) > 1 {
			t.Error("Unexpected N of peers: > 1")
		}
	}

	h0Info := h[0].Peerstore().PeerInfo(h[0].ID())
	h0Info.Addrs = append(h0Info.Addrs, h[0].Addrs()...)

	h[1].testPeer(h0Info)
	h[1].savePeer(h0Info)

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
func startRandomHost(p int) AkhHost {
	private, _, _ := blockchain.NewKeys()
	privateBytes, _ := crypto.MarshalPrivateKey(private)
	return StartHost(p, privateBytes, false)
}
