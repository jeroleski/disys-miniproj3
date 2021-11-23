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

const (
	port = ":8080"
)

var serverAddr string
var serverid int64 = 0
var currentHighestBid int64 = 0

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

var peers []pb.AuctionServiceClient = make([]pb.AuctionServiceClient, 0)
var ch *ConnectionHolder = &ConnectionHolder{connectedClients: make(map[string]chan BidInfo, 0)}
var hb *HighestBid = &HighestBid{currentHighestBid: 0, user: ""}

func main() {
	//Server listens on the server port and handles error.
	lis, err1 := net.Listen("tcp", port)
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

func Port(NodeId int64) string {
	file, err := os.Open("ServerPorts.txt")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var Port0 string
	for scanner.Scan() {
		IdPort := strings.Split(scanner.Text(), " ")
		Id, _ := strconv.ParseInt(IdPort[0], 10, 64)
		if Id == NodeId {
			Port0 = IdPort[1]
		}
	}
	return Port0

}

func (s *AuctionServiceServer) MakeBid(ctx context.Context, Bid *pb.Bid) (*pb.Response, error) {
	AddClient(Bid.User)
	success := SetBid(Bid.Amount, Bid.User)
	if !success {
		return &pb.Response{Ack: "nono"}, nil
	}

	for _, peer := range peers {
		peer.UpdateHighestBid(ctx, Bid)
	}

	log.Printf("%s made a bid of $%d", Bid.User, Bid.Amount)
	BroadcastBid(NewBidInfo(Bid.Amount, Bid.User))

	return &pb.Response{Ack: "yaya"}, nil
}

func (s *AuctionServiceServer) GetCurrentInfo(ctx context.Context, Request *pb.Request) (*pb.Bid, error) {
	log.Println("tries to listen")
	c := GetChannel(Request.User)
	bidInfo := <-c
	return &pb.Bid{Amount: bidInfo.Amount, User: bidInfo.User}, nil
}

func (s *AuctionServiceServer) Result(ctx context.Context, Request *pb.Void) (*pb.Bid, error) {
	return nil, nil
}

func (s *AuctionServiceServer) UpdateHighestBid(ctx context.Context, Bid *pb.Bid) (*pb.Response, error) {
	hb = &HighestBid{currentHighestBid: Bid.Amount, user: Bid.User}
	return &pb.Response{Ack: "yaya"}, nil
}

func BroadcastBid(bidInfo BidInfo) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for _, c := range ch.connectedClients {
		go Send(bidInfo, c)
	}
}

func Send(bidInfo BidInfo, c chan BidInfo) {
	c <- bidInfo
}

type BidInfo struct {
	Amount int32
	User   string
}

func NewBidInfo(amount int32, user string) BidInfo {
	return BidInfo{Amount: amount, User: user}
}

type ConnectionHolder struct {
	connectedClients map[string](chan BidInfo)
	mu               sync.Mutex
}

func AddClient(User string) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.connectedClients[User] == nil {
		ch.connectedClients[User] = make(chan BidInfo, 0)
		return true
	}

	return false
}
func GetChannel(User string) chan BidInfo {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.connectedClients[User]
}

type HighestBid struct {
	currentHighestBid int32
	user              string
	mu                sync.Mutex
}

func SetBid(Amount int32, User string) bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.currentHighestBid > Amount {
		return false
	}

	hb.currentHighestBid = Amount
	hb.user = User

	return true
}
func GetBid() (int32, string) {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	return hb.currentHighestBid, hb.user
}
