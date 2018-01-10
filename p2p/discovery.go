package p2p

import (
	"bufio"
	"fmt"
	"log"
	"github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"strings"
	"os"
)

const HostsInfoPath = "/tmp/akhhosts.info"

func (h *AkhHost) DiscoverPeers() {
	peers := readHostsInfo()
	log.Printf("### pre-defined peers number = %d", len(peers))
	h.populatePeerStore(peers)
}

//TODO validation and error handling
func (h *AkhHost) AddPeerManually(remotePeerAddr string, remotePeerID string) {
	split := strings.Split(remotePeerAddr, ":")
	addrStr := fmt.Sprintf("/ip4/%s/tcp/%s", split[0], split[1])
	peerInfo := newPeerInfo(addrStr, remotePeerID)
	h.populatePeerStore([]ps.PeerInfo{peerInfo})
}

func newPeerInfo(addrStr string, remotePeerID string) ps.PeerInfo {
	addr, _ := ma.NewMultiaddr(addrStr)
	decID, err := peer.IDB58Decode(remotePeerID)
	if err != nil {
		log.Fatal(err)
	}
	peerInfo := ps.PeerInfo{ID: decID, Addrs: []ma.Multiaddr{addr}}
	return peerInfo
}

func readHostsInfo() []ps.PeerInfo {

	peers := make([]ps.PeerInfo, 0, 10) //magic constant

	file, err := os.Open(HostsInfoPath)
	if err != nil {
		log.Print(err)
		return peers
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ":")
		peers = append(peers, newPeerInfo(split[0], split[1]))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return peers
}

func (h *AkhHost) DumpHostInfo() (err error) {
	info := fmt.Sprintf("%s:%s\n", h.Addrs()[0], h.ID().Pretty())
	f, err := os.OpenFile(HostsInfoPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error while dumping host info to local registry: %s\n", err)
		return
	}

	defer f.Close()

	_, err = f.WriteString(info)
	return
}

//TODO 1) check connectivity and delete invalid peers
// 2) request valid peers for there peers
func (h *AkhHost) populatePeerStore(peerInfos []ps.PeerInfo) {
	for _, peerInfo := range peerInfos {
		h.AddPeer(&peerInfo)
	}
}

func (h *AkhHost) AddPeer(peerInfo *ps.PeerInfo) {
	h.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)
}
