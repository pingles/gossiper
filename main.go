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

// return a slice of byte arrays containing messages to broadcast

// overhead is amount of overhead in packet, limit is total size. total
// message size must be smaller than limit
func (m *GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	empty := make([][]byte, 0)
	return empty
}

// notified when we receive a message. blocking is bad as it would block
// the whole udp loop. also, byte slice may be modified so a copy should
// be taken.
func (m *GossipDelegate) NotifyMsg(msg []byte) {
	log.Println("delegate: notify msg")
}


// send local state to remote side, bool indicates as a
// result of a join
func (m *GossipDelegate) LocalState(join bool) []byte {
	log.Println("delegate: local state, joining", join)
	empty := make([]byte, 0)
	return empty
}

// merge remote side state
func (m *GossipDelegate) MergeRemoteState(s []byte, join bool) {
	log.Println("delegate: merge remote state")
}

// provide meta-data used when broadcasting an alive message
func (m *GossipDelegate) NodeMeta(limit int) []byte {
	log.Println("delegate: node meta")
	empty := make([]byte, 0)
	return empty
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

type BindAddr struct {
	Address string
	Port    int
}

func AddressAndPort(s string) (*BindAddr, error) {
	bindParts := strings.Split(s, ":")
	b := &BindAddr{}
	b.Address = bindParts[0]
	
	if len(bindParts) > 1 {
		bindPort, err := strconv.Atoi(bindParts[1])
		if err != nil {
			return nil, err
		}
		b.Port = bindPort
	}
	
	return b, nil
}

func main() {
	hostname, _ := os.Hostname()
	
	var bindAddressAndPort = flag.String("bind", "127.0.0.1:7000", "tcp and udp address for gossip")
	var join = flag.String("join", "", "address of existing node to join to, e.g. 127.0.0.1:7000")
	var name = flag.String("name", hostname, "node name")
	flag.Parse()
	
	binding, err := AddressAndPort(*bindAddressAndPort)
	if err != nil {
		log.Fatal(err)
	}
	
	config := ml.DefaultLocalConfig()
	config.BindAddr = binding.Address
	if binding.Port > 0 {
		config.BindPort = binding.Port
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