package main

import (
	"context"
	pb "example/disys-miniproj3/auction"
	"net"

	"google.golang.org/grpc"

	"time"

	"os"
	"strconv"
	"strings"

	"bufio"
	"log"

	bid "example/disys-miniproj3/server/bid"
	timer "example/disys-miniproj3/server/timer"
)

var serverAddr string
var serverid int64 = 0
var currentHighestBid int64 = 0

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

var connections *bid.ConnectionHolder = &bid.ConnectionHolder{ConnectedClients: make(map[string]*bid.BidInfo, 0)}
var highestBid *bid.HighestBidHolder = &bid.HighestBidHolder{BidInfo: &bid.BidInfo{Amount: 69, User: "SELLER"}}
var auctionTimer *timer.Timer = &timer.Timer{Time: time.Second * 10, Await: time.Second * 2, Read: make(map[string](chan time.Duration)), IsTicking: false}

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
	connections.AddClient(Bid.User, highestBid.GetHighestBid())
	success := highestBid.SetBid(Bid.Amount, Bid.User)
	if !success {
		return &pb.Response{Ack: "You must make a minimal bid higher than $" + strconv.FormatInt(int64(highestBid.GetHighestBid().Amount), 10)}, nil
	}

	log.Printf("%s made a bid of $%d", Bid.User, Bid.Amount)
	BroadcastBid(&bid.BidInfo{Bid.Amount, Bid.User})

	return &pb.Response{Ack: "You have made a bid of $" + strconv.FormatInt(int64(highestBid.GetHighestBid().Amount), 10)}, nil
}

func (s *AuctionServiceServer) GetCurrentInfo(ctx context.Context, Request *pb.Request) (*pb.Bid, error) {
	for {
		BidInfo := connections.GetBidInfo(Request.User)
		if BidInfo != nil {
			return &pb.Bid{Amount: BidInfo.Amount, User: BidInfo.User}, nil
		}
	}
}

func (s *AuctionServiceServer) Result(ctx context.Context, Request *pb.Void) (*pb.Bid, error) {
	for {
		if auctionTimer.TimesUp() {
			bid := highestBid.GetHighestBid()
			return &pb.Bid{Amount: bid.Amount, User: bid.User}, nil
		}
	}
}

func (s *AuctionServiceServer) UpdateTime(ctx context.Context, Request *pb.Request) (*pb.Time, error) {
	c := auctionTimer.GetChannel(Request.User)
	for timeLeft := range c {
		s := strconv.Itoa(int(timeLeft.Seconds()))
		return &pb.Time{TimeLeft: s}, nil
	}
	return &pb.Time{TimeLeft: "The auction is over!"}, nil
}

func BroadcastBid(BidInfo *bid.BidInfo) {
	connections.Mu.Lock()
	defer connections.Mu.Unlock()

	for s, _ := range connections.ConnectedClients {
		if s == BidInfo.User {
			continue
		}
		connections.ConnectedClients[s] = BidInfo
	}
}
