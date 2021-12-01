package main

import (
	"context"
	pb "example/disys-miniproj3/auction"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"time"

	"os"
	"strconv"
	"strings"

	"bufio"
	"log"
	"sync"

	bidUtils "example/disys-miniproj3/server/bidUtils"
	timer "example/disys-miniproj3/server/timer"
)

var serverAddr string
var serverid int64 = 0

type AuctionServiceServer struct {
	pb.UnimplementedAuctionServiceServer
}

var bidBroadcaster *bidUtils.BidinfoBroadcaster = &bidUtils.BidinfoBroadcaster{
	UserChannels: make(map[string](chan *bidUtils.BidInfo), 0)}
var highestBid *bidUtils.HighestBidHolder = &bidUtils.HighestBidHolder{
	BidInfo: &bidUtils.BidInfo{Amount: 69, User: "SELLER"}}
var auctionTimer *timer.Timer = &timer.Timer{
	Time:         time.Second * 120,
	Await:        time.Second * 10,
	UserChannels: make(map[string](chan time.Duration)),
	IsTicking:    false,
	OnClose:      func() { bidBroadcaster.CloseAll() }}
var mu sync.Mutex

func init() {
	auctionTimer.OnTick = func() {
		go MakeBackup()
	}
}

func main() {
	//Sets up logs
	//Setup the file for log outputs
	LogFile := "./systemlogs/server.log"
	// open log file
	logFile, err := os.OpenFile(LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			log.Fatalf("File not found: %v\n", err)
		}
	}(logFile)

	log.SetOutput(logFile)

	args := os.Args[1:]
	if len(args) < 1 {
		os.Exit(1)
	}
	Id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Panic(err)
	}
	serverid = Id
	//Server listens on the server port and handles error.
	lis, err1 := net.Listen("tcp", Port((int32(Id))))
	if err1 != nil {
		log.Fatalf("Failed to listen: %v", err1)
	}

	//Create and register a new grpc server
	server := grpc.NewServer()
	pb.RegisterAuctionServiceServer(server, &AuctionServiceServer{})
	log.Printf("Server listening at %v\n", lis.Addr())
	fmt.Printf("Server listening at %v\n", lis.Addr())

	//Connect the port we're listening on with the newly created server.
	if err := server.Serve(lis); err != nil {
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

func (s *AuctionServiceServer) MakeBid(ctx context.Context, bid *pb.Bid) (*pb.Response, error) {
	bidBroadcaster.AddClient(bid.User, highestBid.GetHighestBid())
	success := highestBid.SetBid(bid.Amount, bid.User)
	if !success {
		return &pb.Response{Ack: "You must make a minimal bid higher than $" + strconv.FormatInt(int64(highestBid.GetHighestBid().Amount), 10)}, nil
	}

	log.Printf("%s made a bid of $%d", bid.User, bid.Amount)
	bidBroadcaster.BroadcastToAll(&bidUtils.BidInfo{Amount: bid.Amount, User: bid.User})

	go MakeBackup()

	return &pb.Response{Ack: "You have made a bid of $" + strconv.FormatInt(int64(highestBid.GetHighestBid().Amount), 10)}, nil
}

func (s *AuctionServiceServer) GetStreamHighestbid(request *pb.Request, bidStream pb.AuctionService_GetStreamHighestbidServer) error {
	bidBroadcaster.AddClient(request.User, highestBid.GetHighestBid())
	c := bidBroadcaster.GetChannel(request.User)
	for bid := range c {
		bidStream.Send(&pb.Bid{Amount: bid.Amount, User: bid.User})
	}
	return nil
}

func (s *AuctionServiceServer) Result(ctx context.Context, void *pb.Void) (*pb.Bid, error) {
	for {
		if auctionTimer.TimesUp() {
			bid := highestBid.GetHighestBid()
			return &pb.Bid{Amount: bid.Amount, User: bid.User}, nil
		}
	}
}

func (s *AuctionServiceServer) GetStreamTimeleft(request *pb.Request, timeStream pb.AuctionService_GetStreamTimeleftServer) error {
	auctionTimer.AddClient(request.User)
	c := auctionTimer.GetChannel(request.User)
	for timeLeft := range c {
		time.Sleep(10 * time.Second)
		s := strconv.Itoa(int(timeLeft.Seconds()))
		time := &pb.Time{Msg: s + " seconds left of the auction!"}
		timeStream.Send(time)
	}
	return nil
}

func (s *AuctionServiceServer) ServerBackup(ctx context.Context, backup *pb.Backup) (*pb.Void, error) {
	mu.Lock()
	defer mu.Unlock()
	bid := &bidUtils.BidInfo{Amount: backup.HighestBidAmount, User: backup.HighestBidUser}

	bidBroadcaster = &bidUtils.BidinfoBroadcaster{
		UserChannels: make(map[string](chan *bidUtils.BidInfo), 0)}
	highestBid = &bidUtils.HighestBidHolder{
		BidInfo: bid}
	auctionTimer = &timer.Timer{
		Time:         time.Duration(backup.TimeLeft),
		Await:        time.Second * 10,
		UserChannels: make(map[string](chan time.Duration)),
		IsTicking:    false,
		OnTick:       func() { go MakeBackup() },
		OnClose:      func() { bidBroadcaster.CloseAll() }}

	for _, user := range backup.ConnectedUsers {
		bidBroadcaster.AddClient(user, bid)
		auctionTimer.AddClient(user)
	}

	log.Printf("Backup message recived Bid: %d - user: %s - time: %d \n", backup.HighestBidAmount, backup.HighestBidUser, backup.TimeLeft)

	return &pb.Void{}, nil
}

func MakeBackup() {
	if serverid == 0 {
		conn, err := grpc.Dial(Port(int32(serverid)+1), grpc.WithInsecure())
		if err != nil {
			log.Printf("Did not connect to backup server: %v\n", err)
			return
		}
		defer conn.Close()

		server2 := pb.NewAuctionServiceClient(conn)

		// Contact the server and print out its response.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		bid := highestBid.GetHighestBid()
		_, err = server2.ServerBackup(ctx, &pb.Backup{ConnectedUsers: bidBroadcaster.GetAllUsers(), HighestBidAmount: bid.Amount, HighestBidUser: bid.User, TimeLeft: auctionTimer.GetTimeLeft()})
		if err != nil {
			log.Println("Could not send backup message")
			return
		}
		log.Print("Backup updatet with new bid, user and current time left")
	}
}
