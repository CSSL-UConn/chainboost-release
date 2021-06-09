package blockchain

// import (
// 	"github.com/BurntSushi/toml"
// 	onet "github.com/basedfs"
// 	"github.com/basedfs/log"
// 	"github.com/basedfs/simul/monitor"
// 	"github.com/dedis/cothority/protocols/manage"
// 	//"github.com/dedis/cothority/protocols/manage"
// )

// var magicNum = [4]byte{0xF9, 0xBE, 0xB4, 0xD9}

// func init() {
// 	onet.SimulationRegister("ByzCoinPBFT", NewSimulation)
// 	onet.ProtocolRegisterName("ByzCoinPBFT", func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) { return NewProtocol(n) })
// }

// // Simulation implements sda.Simulation interface
// type Simulation struct {
// 	// sda fields:
// 	onet.SimulationBFTree
// 	// pbft simulation specific fields:
// 	// Blocksize is the number of transactions in one block:
// 	Blocksize int
// }

// // NewSimulation returns a pbft simulation
// func NewSimulation(config string) (onet.Simulation, error) {
// 	sim := &Simulation{}
// 	_, err := toml.Decode(config, sim)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sim, nil
// }

// // Setup implements sda.Simulation interface
// func (e *Simulation) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {
// 	err := EnsureBlockIsAvailable(dir)
// 	if err != nil {
// 		log.Fatal("Couldn't get block:", err)
// 	}

// 	sc := &onet.SimulationConfig{}
// 	e.CreateRoster(sc, hosts, 2000)
// 	err = e.CreateTree(sc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sc, nil
// }

// // Run runs the simulation
// func (e *Simulation) Run(Conf *onet.SimulationConfig) error {
// 	doneChan := make(chan bool)
// 	doneCB := func() {
// 		doneChan <- true
// 	}
// 	// FIXME use client instead
// 	dir := GetBlockDir()
// 	parser, err := NewParser(dir, magicNum)
// 	if err != nil {
// 		log.Error("Error: Couldn't parse blocks in", dir)
// 		return err
// 	}
// 	transactions, err := parser.Parse(0, e.Blocksize)
// 	if err != nil {
// 		log.Error("Error while parsing transactions", err)
// 		return err
// 	}

// 	// FIXME c&p from byzcoin.go
// 	trlist := NewTransactionList(transactions, len(transactions))
// 	header := NewHeader(trlist, "", "")
// 	trblock := NewTrBlock(trlist, header)

// 	// Here we first setup the N^2 connections with a broadcast protocol
// 	pi, err := Conf.Overlay.CreateProtocolSDA("Broadcast", Conf.Tree)
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	proto := pi.(*manage.Broadcast)
// 	// channel to notify we are done
// 	broadDone := make(chan bool)
// 	proto.RegisterOnDone(func() {
// 		broadDone <- true
// 	})

// 	// ignore error on purpose: Start always returns nil
// 	_ = proto.Start()

// 	// wait
// 	<-broadDone
// 	log.Lvl3("Simulation can start!")
// 	for round := 0; round < e.Rounds; round++ {
// 		log.Lvl1("Starting round", round)
// 		p, err := Conf.Overlay.CreateProtocolSDA("ByzCoinPBFT", Conf.Tree)
// 		if err != nil {
// 			return err
// 		}
// 		proto := p.(*Protocol)

// 		proto.trBlock = trblock
// 		proto.onDoneCB = doneCB

// 		r := monitor.NewTimeMeasure("round_pbft")
// 		err = proto.Start()
// 		if err != nil {
// 			log.Error("Couldn't start PrePrepare")
// 			return err
// 		}

// 		// wait for finishing pbft:
// 		<-doneChan
// 		r.Record()

// 		log.Lvl2("Finished round", round)
// 	}
// 	return nil
// }
