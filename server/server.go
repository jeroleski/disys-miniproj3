package main

import (
	"net"
	"context"
	"google.golang.org/grpc"
	pb "example/disys-miniproj3/auction"


	"os"
	"strconv"
	"strings"

	"bufio"
	"fmt"
	"log"

	"sync"
)

const (
	port = ":8008"
)

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

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
		pier := setupNewPiernode()
		piers = append(piers, pier)

		log.Fatalf("Failed to serve: %v", err2)
	}
}

var piers []*pb.AuctionServiceClient = make([]*pb.AuctionServiceClient, 0)
func setupNewPiernode() *pb.AuctionServiceClient {
	serverAddr = Port(Id)
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
			SetUpClient(serverid)
		}
	}(conn)

	client := pb.NewAuctionServiceClient(conn)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return &client
}

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


func (s *AuctionServiceServer) MakeBid(ctx context.Context, bid *pb.bid) *pb.response, error {
	succes := highestBid.SetBid(bid.amount, bid.user)
	if ! success {
		return &pb.Response{"nono"}, nil
	}

	for _, pier := range piers {
		pier.UpdateHighestBid(ctx, bid)
	}

	return &pb.Response{"yaya"}, nil
}

func (s *AuctionServiceServer) Result(ctx context.Context, request *pb.request) *pb.outcome, error {

}

func (s *AuctionServiceServer) UpdateHighestBid(ctx context.Context, request *pb.request) *pb.outcome, error {
	highestBid &HighestBid{currentHighestBid: bid.amount, User: bid.user}
	return &pb.Response{"yaya"}, nil
}


var connectionHolder *ConnectionHolder = &ConnectionHolder{connectedClients: make(chan int, 0)}
type ConnectionHolder struct {
	connectedClients [string](chan int)
	mu sync.Mutex
}

func (ch *ConnectionHolder) AddClient(user string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.connectedClients[user] == nil {
		ch.connectedClients[user] = make(chan int)
	}
	
}

func (ch *ConnectionHolder) GetChannel(user string) chan int {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.connectedClients[user]
}




var highestBid *HighestBid = &HighestBid{currentHighestBid: 0, User: nil}
type HighestBid struct {
	currentHighestBid int
	user string
	mu sync.Mutex
}

func (hb *HighestBid) SetBid(amount int, user string) bool {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if hb.highestBid > amount {
		return false
	}

	hb.currentHighestBid = amount
	hb.user = user

	return true
}