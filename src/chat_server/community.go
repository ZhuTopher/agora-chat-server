package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

type Community struct {
	// public fields
    ID          string     // corresponds to neighbourhood (i.e: uWaterloo)
    Clients 	map[uint32]*Client

    caChan 		chan *ClientAction
    done 		chan bool
    loopWG		sync.WaitGroup
}

func NewComm(id string) (comm *Community) {
	comm = new(Community)
	comm.ID = id
	comm.Clients = make(map[uint32]*Client)
	comm.caChan = make(chan *ClientAction)
	comm.done = make(chan bool)

	comm.loopWG.Add(1)
    go comm.controlLoop()

	return
}

// Will disconnect all Clients in this Community
func (comm *Community) Shutdown() (err error) {
    // stop comm loops from processing
    close(comm.done) // sends on channel to all receivers
    comm.loopWG.Wait()

    // closes all client connections in comm.Clients
    var wg sync.WaitGroup
    for _, cPtr := range comm.Clients {
        wg.Add(1)
        go func(wg *sync.WaitGroup, c *Client) {
            defer wg.Done()
            c.Disconnect()
        }(&wg, cPtr)
    }
    log.Printf("Community %v disconnecting all clients...\n", comm.ID)
    wg.Wait()

    return
}


func (comm *Community) controlLoop() {
    defer func() {
        log.Printf("Community %s exiting controlLoop()\n", comm.ID)
        comm.loopWG.Done()
    }()

ControlLoop:
    for {
        select {
        case <-comm.done:
            break ControlLoop
        case caPtr := <-comm.caChan:
        	log.Println("comm.controlLoop(): Received a Client Action")
            comm.handleCA(caPtr)
        }
    }

    // deferred comm.loopWG.Done() called
}

// method implementations in client_actions.go
func (comm *Community) handleCA(caPtr *ClientAction) {
	// TODO: impl
}

func (comm *Community) AddClient(c *Client) error {
	if _, ok := comm.Clients[c.ID]; ok {
		return errors.New(fmt.Sprintf(
			"Client ID %v already exists in community %s\n",
			c.ID, comm.ID))
	}

	comm.Clients[c.ID] = c

	return nil
}

func (comm *Community) RemoveClient(c *Client) error {
	if _, ok := comm.Clients[c.ID]; ok {
		return errors.New(fmt.Sprintf(
			"Client ID %v DNE in community %s\n",
			c.ID, comm.ID))
	}

	delete(comm.Clients, c.ID)

	return nil
}