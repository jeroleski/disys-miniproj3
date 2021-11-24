package bid

import (
	"sync"
)

type ConnectionHolder struct {
	ConnectedClients map[string]*BidInfo
	Mu               sync.Mutex
}

type BidInfo struct {
	Amount int32
	User   string
}

type HighestBidHolder struct {
	BidInfo *BidInfo
	Mu      sync.Mutex
}

func (ch *ConnectionHolder) AddClient(user string, highestBid *BidInfo) bool {
	ch.Mu.Lock()
	defer ch.Mu.Unlock()

	if ch.ConnectedClients[user] == nil {
		ch.ConnectedClients[user] = highestBid
		return true
	}

	return false
}
func (ch *ConnectionHolder) GetBidInfo(user string) *BidInfo {
	ch.Mu.Lock()
	defer ch.Mu.Unlock()

	BidInfo := ch.ConnectedClients[user]
	ch.ConnectedClients[user] = nil
	return BidInfo
}

func (ch *ConnectionHolder) BroadcastBid(bidInfo *BidInfo) {
	ch.Mu.Lock()
	defer ch.Mu.Unlock()

	for s, _ := range ch.ConnectedClients {
		if s == bidInfo.User {
			continue
		}
		ch.ConnectedClients[s] = bidInfo
	}
}

func (hb *HighestBidHolder) SetBid(Amount int32, User string) bool {
	hb.Mu.Lock()
	defer hb.Mu.Unlock()

	if hb.BidInfo.Amount > Amount {
		return false
	}

	hb.BidInfo = &BidInfo{Amount: Amount, User: User}

	return true
}
func (hb *HighestBidHolder) GetHighestBid() *BidInfo {
	hb.Mu.Lock()
	defer hb.Mu.Unlock()

	return &BidInfo{Amount: hb.BidInfo.Amount, User: hb.BidInfo.User}
}
