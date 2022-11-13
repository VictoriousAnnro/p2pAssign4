package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	ping "github.com/VictoriousAnnro/p2pAssign4/grpc"
	"google.golang.org/grpc"
)

type Status string

const (
	Free   = "Free"
	Wanted = "Wanted"
	Held   = "Held"
)

//go run main.go 0
//go run main.go 1
//go run main.go 2

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:      ownPort,
		queue:   make([]ping.PingClient, 0),
		clients: make(map[int32]ping.PingClient),
		ctx:     ctx,
		state:   Free,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	ping.RegisterPingServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := ping.NewPingClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		p.sendRequestToAll()
	}
}

type peer struct {
	ping.UnimplementedPingServer
	id      int32
	lamport int32
	queue   []ping.PingClient
	clients map[int32]ping.PingClient
	ctx     context.Context
	state   Status
}

func (p *peer) RequestCar(ctx context.Context, req *ping.Request) (*ping.Reply, error) {

	for {
		if p.state == Wanted && (p.lamport < req.Lamport || (p.id < req.Id && p.lamport == req.Lamport)) || p.state == Held {
			//put in queue, reply later
			//queue.add(clinet)
			time.Sleep(100 * time.Millisecond)
			continue
		} else {
			break
		}
	}

	if p.lamport < req.Lamport {
		p.lamport = req.Lamport
	}

	p.lamport++

	rep := &ping.Reply{Lamport: p.lamport}
	return rep, nil
}

func (p *peer) sendRequestToAll() {
	p.state = Wanted
	p.lamport++
	request := &ping.Request{Id: p.id,
		Lamport: p.lamport}

	for id, client := range p.clients {
		reply, err := client.RequestCar(p.ctx, request)
		if err != nil {
			fmt.Println("something went wrong")
		}
		fmt.Printf("Got reply from id %v: %v\n", id, reply)
	}

	p.state = Held
	p.CriticalSection()
}

func (p *peer) CriticalSection() {
	p.lamport++

	log.Printf("Client %v is playing Ceenja Impact", p.id)
	time.Sleep(20000 * time.Millisecond)
	log.Printf("Client %v is done", p.id)
	p.state = Free
}
