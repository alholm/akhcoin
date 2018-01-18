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
	"github.com/libp2p/go-libp2p-protocol"
	"context"
)

const HostsInfoPath = "/tmp/akhhosts.info"

func (h *AkhHost) DiscoverPeers() {
	peers := readHostsInfo()
	log.Printf("DEBUG: pre-defined peers number = %d", len(peers))
	h.populatePeerStore(peers)
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
		peerInfo, err := newPeerInfo(split[0], split[1])
		if err == nil {
			peers = append(peers, peerInfo)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return peers
}

//TODO 1) check connectivity and delete invalid peers
//	   2) parallelize
func (h *AkhHost) populatePeerStore(peerInfos []ps.PeerInfo) {
	log.Println("DEBUG: populating peerstore...")

	for _, peerInfo := range peerInfos {
		err := h.addPeer(peerInfo)
		if err != nil {
			log.Println(fmt.Errorf("Error while populating %s peerstore: %s\n", h.ID(), err))
			continue
		}

		peerPeers, _ := h.askForPeers(peerInfo.ID)
		for _, peerPeerInfo := range peerPeers {
			log.Printf("DEBUG: Received peer: %s\n", peerPeerInfo.ID.Pretty())
			err = h.addPeer(peerPeerInfo)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (h *AkhHost) addPeer(peerInfo ps.PeerInfo) (err error) {
	err = h.Connect(context.Background(), peerInfo)
	if err == nil {
		h.Peerstore().SetAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)
	}
	return
}

func (h *AkhHost) askForPeers(peerID peer.ID) (peerInfos []ps.PeerInfo, err error) {
	log.Printf("DEBUG: %s asking for peers from %s\n", h.ID(), peerID)
	var idAddrMap map[string]string

	err = h.ask(peerID, GetPeersMessage{}, DiscoverProto, &idAddrMap)

	for id, addr := range idAddrMap {
		peerInfo, peerErr := newPeerInfo(addr, id)
		if peerErr != nil {
			log.Printf("Error adding peer %s, %s: %s\n", id, addr, peerErr)
		}
		peerInfos = append(peerInfos, peerInfo)
	}
	return
}

type GetPeersMessage struct {
	Message
}

func newPeerInfo(addrStr string, remotePeerID string) (peerInfo ps.PeerInfo, err error) {
	addr, err := ma.NewMultiaddr(addrStr)
	if err != nil {
		return
	}
	decID, err := peer.IDB58Decode(remotePeerID)
	if err != nil {
		return
	}
	peerInfo = ps.PeerInfo{ID: decID, Addrs: []ma.Multiaddr{addr}}
	return
}

type DiscoverStreamHandler struct {
	store *ps.Peerstore
}

func (*DiscoverStreamHandler) protocol() protocol.ID {
	return DiscoverProto
}

func (drp *DiscoverStreamHandler) handle(ws *WrappedStream) {

	getAnswer := func() interface{} {
		peerIDs := (*drp.store).Peers()
		infos := make(map[string]string, len(peerIDs))
		for _, id := range peerIDs {
			if id == ws.stream.Conn().RemotePeer() {
				continue
			}
			addrs := (*drp.store).Addrs(id)
			//localhost has no self addrs
			if len(addrs) > 0 {
				infos[id.Pretty()] = addrs[0].String()
			}
		}
		return infos
	}

	err := answer(ws, &GetPeersMessage{}, getAnswer)
	if err != nil {
		log.Printf("Error handling discover stream: %s", err)
	}
}

//remotePeerAddr format: <dot-separated IPv4>:<post>, for example: 127.0.0.1:9000
//remotePeerID  - unprettyfied ID
//TODO validation and error handling
func (h *AkhHost) AddPeerManually(remotePeerAddr string, remotePeerID string) (err error) {
	split := strings.Split(remotePeerAddr, ":")
	addrStr := fmt.Sprintf("/ip4/%s/tcp/%s", split[0], split[1])
	peerInfo, err := newPeerInfo(addrStr, remotePeerID)
	if err == nil {
		h.populatePeerStore([]ps.PeerInfo{peerInfo})
	}
	return
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
