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
	"fmt"
	"log"

	"sync"
)

const (
	port = ":8080"
)

var serverAddr string
var serverid int64 = 0
var currentHighestBid int64

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

var peers []pb.AuctionServiceClient = make([]pb.AuctionServiceClient, 0)
var ch *ConnectionHolder = &ConnectionHolder{connectedClients: make(map[string]chan int32, 0)}
var hb *HighestBid = &HighestBid{currentHighestBid: 0, User: ""}

func main() {
	//Server listens on the server port and handles error.
	lis, err1 := net.Listen("tcp", port)
	if err1 != nil {
		log.Fatal("Failed to listen: %v", err1)
	}

	//Create and register a new grpc server
	server := grpc.NewServer()
	pb.RegisterAuctionServiceServer(server, &AuctionServiceServer{})
	fmt.Printf("Server listening at %v\n", lis.Addr())

	//Connect the port we're listening on with the newly created server.
	err2 := server.Serve(lis)
	if err2 != nil {
		//peer := setupNewPeerNode()
		//peers = append(peers, peer)

		log.Fatalf("Failed to serve: %v", err2)
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
	succes := SetBid(Bid.Amount, Bid.User)
	if !succes {
		return &pb.Response{Ack: "nono"}, nil
	}

	for _, peer := range peers {
		peer.UpdateHighestBid(ctx, Bid)
	}

	BroadcastBid(Bid.Amount)

	return &pb.Response{Ack: "yaya"}, nil
}

func (s *AuctionServiceServer) GetHighestBid(ctx context.Context, Request *pb.Request) (*pb.Bid, error) {
	c := GetChannel(Request.User)
	highestBid := <-c
	return &pb.Bid{Amount: highestBid, User: ""}, nil
}

func (s *AuctionServiceServer) Result(ctx context.Context, Request *pb.Void) (*pb.Bid, error) {
	return nil, nil
}

func (s *AuctionServiceServer) UpdateHighestBid(ctx context.Context, Bid *pb.Bid) (*pb.Response, error) {
	hb = &HighestBid{currentHighestBid: Bid.Amount, User: Bid.User}
	return &pb.Response{Ack: "yaya"}, nil
}

func BroadcastBid(Amount int32) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for _, c := range ch.connectedClients {
		go Send(Amount, c)
	}
}

func Send(Amount int32, c chan int32) {
	c <- Amount
}

type ConnectionHolder struct {
	connectedClients map[string](chan int32)
	mu               sync.Mutex
}

func AddClient(User string) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.connectedClients[User] == nil {
		ch.connectedClients[User] = make(chan int32)
		return true
	}

	return false
}

func GetChannel(User string) chan int32 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.connectedClients[User]
}

type HighestBid struct {
	currentHighestBid int32
	User              string
	mu                sync.Mutex
}

func SetBid(Amount int32, User string) bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.currentHighestBid > Amount {
		return false
	}

	hb.currentHighestBid = Amount
	hb.User = User

	return true
}
