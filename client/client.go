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

var client pb.AuctionServiceClient
var ctx context.Context
var user string

var serverId int32 = 0

var closeReadBids chan bool
var closeBidStream chan bool
var closeTimeStream chan bool

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
	conn, err := grpc.Dial(Port(serverId), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("connection problem : %v", err)
			Reconnect()
		}
	}(conn)

	client = pb.NewAuctionServiceClient(conn)

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Println("Please write your user name:")

	fmt.Scan(&user)
	log.Printf("User %s has connected to the auction\n", user)

	StartClient()
}

func Reconnect() {
	log.Print("Reconnecting")
	serverId++
	conn, err := grpc.Dial(Port(serverId), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("connection problem : %v", err)
		}
	}(conn)

	client = pb.NewAuctionServiceClient(conn)

	StartClient()
}

func StartClient() {
	closeReadBids = make(chan bool)
	closeBidStream = make(chan bool)
	closeTimeStream = make(chan bool)

	go ListenForTime(closeTimeStream)
	go ListenForBids(closeBidStream)
	go MakeBids(closeReadBids)
	over := GetResult()

	go CloseRoutine(closeReadBids)
	go CloseRoutine(closeBidStream)
	go CloseRoutine(closeTimeStream)

	if !over {
		Reconnect()
	}
}

func CloseRoutine(c chan bool) {
	c <- true
}

func ListenForTime(quit chan bool) {
	timeStream, err := client.GetStreamTimeleft(ctx, &pb.Request{User: user})
	if err != nil {
		log.Print("Could not get time client\n", err)
		return
	}

	for {
		select {
		case <-quit:
			return
		default:
		}

		time, err := timeStream.Recv()
		if err != nil {
			break
		}

		log.Println(time.Msg)
	}
}

func ListenForBids(quit chan bool) {
	bidStream, err := client.GetStreamHighestbid(ctx, &pb.Request{User: user})
	if err != nil {
		log.Print("Could not get bid client\n", err)
		return
	}

	for {
		select {
		case <-quit:
			return
		default:
		}

		bid, err := bidStream.Recv()
		if err != nil {
			break
		}

		log.Printf("%s has bid $%d on the auction!\n", bid.User, bid.Amount)
	}
}

func MakeBids(quit chan bool) {
	for {
		select {
		case <-quit:
			return
		default:
		}

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
			break
		}

		log.Println(response.Ack)
	}
}

func GetResult() bool {
	bid, err := client.Result(ctx, &pb.Void{})
	if err != nil {
		log.Printf("Could not get Result: %v\n", err)
		return false
	}

	if bid.User == user {
		bid.User = "You"
	}

	log.Printf("%s have bought \"SULFURAS, HAND OF RAGNAROS\" for $%v\n", bid.User, bid.Amount)
	if bid.User == "You" {
		log.Println("Please call in to give us your credit card number!")
	}

	return true
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
