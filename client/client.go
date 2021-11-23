package main

import (
	"context"
	pb "example/disys-miniproj3/auction"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	serverAddr = "localhost:8080"
)

var client pb.AuctionServiceClient
var ctx context.Context

var user string

func main() {
	//Sets up logs
	//Setup the file for log outputs
	LogFile := "./systemlogs/client.log"
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

	//log.SetOutput(logFile)
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("connection problem: %v", err)
		}
	}(conn)

	client = pb.NewAuctionServiceClient(conn)

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Println("Please write your user name:")

	fmt.Scan(&user)
	log.Printf("User %s has connected to the auction\n", user)

	go Listen()
	ReadBids()
	//Result()
}

func Result() {
	bid, err := client.Result(ctx, &pb.Void{})
	if err != nil {
		log.Printf("Could not listen for a result: %v\n", err)
	}
	log.Printf("AND THE WINNER IS:\n%s with a bid of %d\n", bid.User, bid.Amount)
}

func ReadBids() {
	for {
		var input string
		fmt.Scan(&input)

		a, err1 := strconv.Atoi(input)
		if err1 != nil {
			log.Printf("%s is not a convertible value\n", input)
			continue
		}

		amount := int32(a)
		response, err2 := client.MakeBid(ctx, &pb.Bid{Amount: amount, User: user})
		if err2 != nil {
			log.Fatalf("Could not make a bid: %v\n", err2)
		}

		log.Println(response.Ack)
	}
}

func Listen() {
	for {
		currentHighestBid, err := client.GetCurrentInfo(ctx, &pb.Request{User: ""})
		if err != nil {
			log.Fatalf("Could not get Info\n", err)
		}

		log.Printf("'%s' has bid $%d on the item!\n", currentHighestBid.User, currentHighestBid.Amount)
	}
}

func Port(NodeId int32) string {
	file, err := os.Open("ServerPorts.txt")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var Port0 string
	for scanner.Scan() {
		IdPort := strings.Split(scanner.Text(), " ")
		Id, _ := (strconv.ParseInt(IdPort[0], 10, 64))
		if int32(Id) == NodeId {
			Port0 = IdPort[1]
		}
	}
	return Port0

}
