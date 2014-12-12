package main

import (
	"flag"
	"log"
	ml "github.com/hashicorp/memberlist"
	"os/signal"
	"os"
	"strings"
	"strconv"
	"time"
)

type Gossiper struct {
	shutdownCh chan bool
	memberlist *ml.Memberlist
}

func NewGossiper(c *ml.Config) (*Gossiper, error) {
	list, err := ml.Create(c)
	if err != nil {
		return nil, err
	}
	stopCh := make(chan bool)
	return &Gossiper{stopCh, list}, nil
}

// waits for a shutdown hook from shutdownCh
func (g *Gossiper) Wait() {
	<-g.shutdownCh
}

func (g *Gossiper) PrintMembers() {
	for _, member := range g.memberlist.Members() {
		log.Println("member:", member.Name, member.Addr)
	}
}

func (g *Gossiper) Join(addr string) {
	log.Println("attempting to join", addr)
	g.memberlist.Join([]string{addr})
}

func (g *Gossiper) Leave() {
	defer func() {
		g.shutdownCh<-true
		log.Println("shutdown")
	}()
	timeout, _ := time.ParseDuration("1s")
	err := g.memberlist.Leave(timeout)
	if err != nil {
		log.Println("error leaving", err)
	}
	g.memberlist.Shutdown()
}

func AttachShutdownHandler(g *Gossiper) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			g.Leave()
		}
	}()
}

func main() {
	hostname, _ := os.Hostname()
	
	var bindAddressAndPort = flag.String("bind", "127.0.0.1:7000", "tcp and udp address for gossip")
	var join = flag.String("join", "", "address of existing node to join to, e.g. 127.0.0.1:7000")
	var name = flag.String("name", hostname, "node name")
	flag.Parse()
	
	config := ml.DefaultLocalConfig()
	bindParts := strings.Split(*bindAddressAndPort, ":")
	config.BindAddr = bindParts[0]
	if len(bindParts) > 1 {
		bindPort, err := strconv.Atoi(bindParts[1])
		if err != nil {
			log.Fatal("invalid bind address", err)
		}
		config.BindPort = bindPort
	}
	config.Name = *name
	
	gossiper, err := NewGossiper(config)
	if err != nil {
		log.Fatal(err)
	}
	AttachShutdownHandler(gossiper)
	
	if *join != "" {
		gossiper.Join(*join)
	}
	
	gossiper.PrintMembers()
	gossiper.Wait()
}