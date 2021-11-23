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
var ch *ConnectionHolder = &ConnectionHolder{connectedClients: make(map[string]chan int, 0)}
var hb *HighestBid = &HighestBid{currentHighestBid: 0, user: ""}

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

func (s *AuctionServiceServer) MakeBid(ctx context.Context, bid *pb.bid) (*pb.response, error) {
	AddClient(bid.user)
	succes := SetBid(bid.amount, bid.user)
	if !succes {
		return &pb.response{"nono"}, nil
	}

	for _, peer := range peers {
		peer.UpdateHighestBid(ctx, bid)
	}

	BroadcastBid(bid.amount)

	return &pb.response{ack: "yaya"}, nil
}

func (s *AuctionServiceServer) GetHighestBid(ctx context.Context, request *pb.request) (*pb.bid, error) {
	c := GetChannel(request.user)
	highestBid := <-c
	return &pb.bid{amount: highestBid, user: nil}, nil
}

func (s *AuctionServiceServer) Result(ctx context.Context, request *pb.void) (*pb.bid, error) {

}

func (s *AuctionServiceServer) UpdateHighestBid(ctx context.Context, bid *pb.bid) (*pb.response, error) {
	hb = &HighestBid{currentHighestBid: bid.amount, user: bid.user}
	return &pb.response{ack: "yaya"}, nil
}

func BroadcastBid(amount int) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for _, c := range ch.connectedClients {
		go Send(amount, c)
	}
}

func Send(amount int, c chan int) {
	c <- amount
}

type ConnectionHolder struct {
	connectedClients map[string](chan int)
	mu               sync.Mutex
}

func AddClient(user string) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.connectedClients[user] == nil {
		ch.connectedClients[user] = make(chan int)
		return true
	}

	return false
}

func GetChannel(user string) chan int {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.connectedClients[user]
}

type HighestBid struct {
	currentHighestBid int
	user              string
	mu                sync.Mutex
}

func SetBid(amount int, user string) bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.currentHighestBid > amount {
		return false
	}

	hb.currentHighestBid = amount
	hb.user = user

	return true
}
