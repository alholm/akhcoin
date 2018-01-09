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

const hostsInfoPath = "/tmp/akhhosts.info"

func (h *AkhHost) DiscoverPeers(remotePeerAddr string, remotePeerID string) {
	peers := readHostsInfo()
	log.Printf("### pre-defined peers number = %d", len(peers))
	if len(remotePeerAddr) > 0 && len(remotePeerID) > 0 {
		split := strings.Split(remotePeerAddr, ":")
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%s", split[0], split[1])
		peerInfo := newPeerInfo(addrStr, remotePeerID)
		peers = append(peers, peerInfo)
	}

	h.populatePeerStore(peers)
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

	file, err := os.Open(hostsInfoPath)
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

func (h *AkhHost) DumpHostInfo(host AkhHost) error {
	info := fmt.Sprintf("%s:%s\n", host.Addrs()[0], host.ID().Pretty())
	fmt.Print(info)
	f, err := os.OpenFile(hostsInfoPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(info)
	return err
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
