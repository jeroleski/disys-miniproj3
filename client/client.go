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

var serverAddr string
var serverid int64 = 0
var currentHighestBid int64

var client pb.AuctionServiceClient
var ctx context.Context

func main() {

	SetUpLog()
	//Currently takes id 0 because its the first server
	SetUpClient(serverid)

	go Result()

	Bid()

}

func Result(){
	time.Sleep(time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for{
		time.Sleep(time.Second*1)
		respons, err := client.Result(ctx,&pb.Void{})
		if err != nil {
			log.Fatalf("Could not get Result", err)
		}
		if(int64(respons.Amount) > currentHighestBid){
			
			fmt.Printf("New bid %b\n")
		}
		


	}
}












func SetUpLog(){
	//Setup the file for log outputs
	LogFile := "./systemlogs/node.log"
	// open log file
	logFile, err := os.OpenFile(LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			log.Fatalf("File not found: %v", err)
		}
	}(logFile)

	log.SetOutput(logFile)	
}

func SetUpClient(Id int64) {
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

	client = pb.NewAuctionServiceClient(conn)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
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
