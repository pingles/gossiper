package main

import (
	"flag"
	"log"
	ml "github.com/hashicorp/memberlist"
	"os/signal"
	"os"
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
	var bind = flag.String("bind", "127.0.0.1", "tcp and udp address for gossip")
	config := ml.DefaultLocalConfig()
	config.BindAddr = *bind
	
	gossiper, err := NewGossiper(config)
	if err != nil {
		log.Fatal(err)
	}
	AttachShutdownHandler(gossiper)
	
	gossiper.PrintMembers()
	gossiper.Wait()
}