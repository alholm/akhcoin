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
	"time"
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

type TestedPeer struct {
	ps.PeerInfo
	err error
}

//TODO 1) delete invalid peers
func (h *AkhHost) populatePeerStore(peerInfos []ps.PeerInfo) {
	log.Println("DEBUG: populating peerstore...")

	peerCh := make(chan TestedPeer)
	countCh := make(chan int)

	go getPeers(h, peerInfos, 1, peerCh, countCh)

	counter, expected, processed := 0, 0, 0
	done := make(chan bool)

	timeout := time.After(5 * time.Second)
	for {
		select {
		case delta := <-countCh:
			if delta > 0 {
				expected += delta
			}
			if counter += delta; counter == 0 {
				go func() { done <- true }()
			}
		case testedPeer := <-peerCh:
			processed++
			if testedPeer.err == nil {
				log.Printf("DEBUG populatePeerStore: Received peer: %s\n", testedPeer.ID.Pretty())
				h.addPeer(testedPeer.PeerInfo)
			} else {
				log.Println(fmt.Errorf("Error while adding peer %s to peerstore: %s\n", h.ID(), testedPeer.err))
			}
		case <-done:
			if expected == processed {
				return
			}
			//let peers that are late to be processed
			go func() { time.Sleep(10 * time.Millisecond); done <- true }()

		case <-timeout:
			log.Printf("DEBUG populatePeerStore: %d of expected %d peers collected, exited by timeout\n", processed, expected)
			return
		}
	}

}
func getPeers(h *AkhHost, peerInfos []ps.PeerInfo, depth int, ch chan TestedPeer, countCh chan int) {
	//how many peers we're about to test and store
	countCh <- len(peerInfos)
	for _, peerInfo := range peerInfos {
		go func() {
			//one peer processed for sure, and in case it has other peers to process, balance (counter) will be > 0,
			//as this function recursive call already sent len(peerInfos) to counterCh
			defer func() { countCh <- -1 }()

			testedPeer := TestedPeer{peerInfo, h.testPeer(peerInfo)}
			ch <- testedPeer

			if depth != 0 {
				peerPeers, _ := h.askForPeers(peerInfo.ID)
				getPeers(h, peerPeers, depth-1, ch, countCh)
			}
		}()

	}

}

func (h *AkhHost) addPeer(peerInfo ps.PeerInfo) {
	//h.testPeer(peerInfo)
	h.Peerstore().SetAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)
}
func (h *AkhHost) testPeer(peerInfo ps.PeerInfo) error {
	return h.Connect(context.Background(), peerInfo)
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
