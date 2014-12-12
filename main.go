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

type GossipDelegate struct {
	gossiper *Gossiper
}

func NewGossipDelegate(g *Gossiper) *GossipDelegate {
	return &GossipDelegate{gossiper: g}
}

func (m *GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	empty := make([][]byte, 0)
	return empty
}

func (m *GossipDelegate) LocalState(join bool) []byte {
	log.Println("delegate: local state")
	empty := make([]byte, 0)
	return empty
}

func (m *GossipDelegate) MergeRemoteState(s []byte, join bool) {
	log.Println("delegate: merge remote state")
}

func (m *GossipDelegate) NodeMeta(limit int) []byte {
	log.Println("delegate: node meta")
	empty := make([]byte, 0)
	return empty
}

func (m *GossipDelegate) NotifyMsg(msg []byte) {
	log.Println("delegate: notify msg")
}

//////////////////




func NewGossiper(c *ml.Config) (*Gossiper, error) {
	stopCh := make(chan bool)
	gossiper := &Gossiper{shutdownCh: stopCh}
	delegate := NewGossipDelegate(gossiper)
	c.Delegate = delegate
	
	list, err := ml.Create(c)
	if err != nil {
		return nil, err
	}
	gossiper.memberlist = list
	
	return gossiper, nil
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