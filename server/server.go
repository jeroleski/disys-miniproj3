package main

import (
	"context"
	pb "example/disys-miniproj3/auction"
	"net"

	"google.golang.org/grpc"

	//"time"

	"os"
	"strconv"
	"strings"

	"bufio"
	"log"

	"sync"
)

var serverAddr string
var serverid int64 = 0
var currentHighestBid int64 = 0

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

//var peers []pb.AuctionServiceClient = make([]pb.AuctionServiceClient, 0)
var ch *ConnectionHolder = &ConnectionHolder{connectedClients: make(map[string]*BidInfo, 0)}
var hb *HighestBid = &HighestBid{bidInfo: &BidInfo{Amount: 69, User: "SELLER"}}

func main() {

	args := os.Args[1:]

	if len(args) < 1 {
		os.Exit(1)
	}
	Id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Panic(err)
	}

	//Server listens on the server port and handles error.
	lis, err1 := net.Listen("tcp", Port((int32(Id))))
	if err1 != nil {
		log.Fatalf("Failed to listen: %v", err1)
	}

	//Create and register a new grpc server
	server := grpc.NewServer()
	pb.RegisterAuctionServiceServer(server, &AuctionServiceServer{})
	log.Printf("Server listening at %v\n", lis.Addr())

	//Connect the port we're listening on with the newly created server.
	if err := server.Serve(lis); err != nil {
		//peer := setupNewPeerNode()
		//peers = append(peers, peer)

		log.Fatalf("Failed to serve: %v", err)
	}
}

/* func setupNewPeerNode() *pb.AuctionServiceClient {
	serverAddr := Port(serverid)
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("connection problem: %v", err)
			serverid++
			log.Fatalf("Trying to reconnect to server: %d", serverid)
			setupNewPeerNode()
		}
	}(conn)

	client := pb.NewAuctionServiceClient(conn)
	var cancel context.CancelFunc
	_, cancel := context.WithTimeout(context.Background(), 10 * time.Minute)
	defer cancel()

	return &client
} */

func Port(ServerId int32) string {
	file, err := os.Open("ServerPorts.txt")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var Port0 string
	for scanner.Scan() {
		IdPort := strings.Split(scanner.Text(), " ")
		Id, _ := strconv.ParseInt(IdPort[0], 10, 64)
		if int32(Id) == ServerId {
			Port0 = IdPort[1]
		}
	}
	return Port0

}

func (s *AuctionServiceServer) MakeBid(ctx context.Context, Bid *pb.Bid) (*pb.Response, error) {
	AddClient(Bid.User)
	success := SetBid(Bid.Amount, Bid.User)
	if !success {
		return &pb.Response{Ack: "You must make a minimal bid higher than $" + strconv.FormatInt(int64(GetHighestBid().Amount), 10)}, nil
	}

	/*for _, peer := range peers {
		peer.UpdateHighestBid(ctx, Bid)
	}*/

	log.Printf("%s made a bid of $%d", Bid.User, Bid.Amount)
	BroadcastBid(&BidInfo{Bid.Amount, Bid.User})

	return &pb.Response{Ack: "You have made a bid of $" + strconv.FormatInt(int64(GetHighestBid().Amount), 10)}, nil
}

func (s *AuctionServiceServer) GetCurrentInfo(ctx context.Context, Request *pb.Request) (*pb.Bid, error) {
	for {
		bidInfo := GetBidInfo(Request.User)
		if bidInfo != nil {
			return &pb.Bid{Amount: bidInfo.Amount, User: bidInfo.User}, nil
		}
	}
}

func (s *AuctionServiceServer) Result(ctx context.Context, Request *pb.Void) (*pb.Bid, error) {
	return nil, nil
}

func (s *AuctionServiceServer) UpdateHighestBid(ctx context.Context, Bid *pb.Bid) (*pb.Response, error) {
	hb = &HighestBid{bidInfo: &BidInfo{Amount: Bid.Amount, User: Bid.User}}
	return &pb.Response{Ack: "yaya"}, nil
}

func BroadcastBid(bidInfo *BidInfo) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for s, _ := range ch.connectedClients {
		if s == bidInfo.User {
			continue
		}
		ch.connectedClients[s] = bidInfo
	}
}

type BidInfo struct {
	Amount int32
	User   string
}

type ConnectionHolder struct {
	connectedClients map[string]*BidInfo
	mu               sync.Mutex
}

func AddClient(user string) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.connectedClients[user] == nil {
		bidInfo := GetHighestBid()
		ch.connectedClients[user] = &BidInfo{Amount: bidInfo.Amount, User: bidInfo.User}
		return true
	}

	return false
}
func GetBidInfo(user string) *BidInfo {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	bidInfo := ch.connectedClients[user]
	ch.connectedClients[user] = nil
	return bidInfo
}

type HighestBid struct {
	bidInfo *BidInfo
	mu      sync.Mutex
}

func SetBid(Amount int32, User string) bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.bidInfo.Amount > Amount {
		return false
	}

	hb.bidInfo = &BidInfo{Amount: Amount, User: User}

	return true
}
func GetHighestBid() *BidInfo {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	return hb.bidInfo
}


/*var timer Timeout
type Timeout struct {
	time string
	mu sync.Mutex
}
func startTime() {
	for range time.Tick(time.Second) {
		timer.mu.Lock()
		do
		timer.mu.Unlock()
	}
}*/

