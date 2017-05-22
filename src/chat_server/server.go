package main

import (
	"errors"
	"fmt"
	"log"
    "sync"
)

// Server takes TCP clients from main and moves them into the
//    respective Community (chat room) according to the chat room id 
type Server struct {
    // public fields
    ID          string     // corresponds to area (i.e: Waterloo)
    Comms       map[string]*Community

    // private fields
    caChan      chan *ClientAction
    running     bool
    done        chan bool
    loopWG      sync.WaitGroup
}

const (
    ROOT_COMM_ID = "root"
)

func NewServer(id string) (s *Server) {
    s = new(Server)

    s.ID = id;
    s.caChan = make(chan *ClientAction)

    s.Comms = make(map[string]*Community)
    s.Comms[ROOT_COMM_ID] = NewComm(ROOT_COMM_ID)

    s.done = make(chan bool)

    if err := s.Start(); err != nil {
        panic("Couldn't start up a NewServer")
    }
    return
}

func (s *Server) Start() (err error) {
    if s.running {
        return errors.New(fmt.Sprintf(
            "Server %s already running.", s.ID))
    }

    s.loopWG.Add(1)
    go s.controlLoop()

    s.running = true
    return
}

// Will disconnect all Clients in s.Comms
func (s *Server) Shutdown() (err error) {
    if !s.running { // TODO: atmomic boolean
        return errors.New(fmt.Sprintf(
            "Server %s already stopped.", s.ID))
    }
    s.running = false

    // stop server loops from processing
    close(s.done) // sends on channel to all receivers
    s.loopWG.Wait()

    // closes all client connections in s.Comms
    var wg sync.WaitGroup
    for _, comm := range s.Comms {
        wg.Add(1)
        go func(wg *sync.WaitGroup, comm *Community) {
            defer wg.Done()
            comm.Shutdown()
            log.Printf("Successfully shutdown Comm %v\n", comm.ID)
        }(&wg, comm)
    }
    wg.Wait()

    return
}

func (s *Server) controlLoop() {
    defer func() {
        log.Printf("Server %s exiting controlLoop()\n", s.ID)
        s.loopWG.Done()
    }()

ControlLoop:
    for {
        select {
        case <-s.done:
            log.Printf("Server %s exiting control loop.\n", s.ID)
            break ControlLoop
        case caPtr := <-s.caChan:
            s.handleCA(caPtr)
        }
    }

    // deferred s.loopWG.Done() called
}

// method implementations in client_actions.go
func (s *Server) handleCA(caPtr *ClientAction) {
    switch caPtr.Action.(type) {
    case JoinServer:
        s.CAJoinServer(caPtr)
    default: // should never happen
        log.Fatalf("(s) Encountered invalid ClientAction: %v\n", (*caPtr))
    }
}


// TODO: just add pointers to Server and approriate Comm instead of ind fields
func (s *Server) AddClientToRootComm(cPtr *Client) (err error) {
    cPtr.CommID = ROOT_COMM_ID
    err = s.AddClient(cPtr)
    if err != nil {
        return err
    }

    return
}

func (s *Server) AddClient(cPtr *Client) (err error) {
    comm, ok := s.Comms[cPtr.CommID]
    if !ok {
        if s.shouldCreateComm(cPtr.CommID) {
            s.Comms[cPtr.CommID] = NewComm(cPtr.CommID)
            comm = s.Comms[cPtr.CommID]
        } else {
            return errors.New(fmt.Sprintf(
                "(s.AddClient) Comm %s DNE", cPtr.CommID))
        }
    }

    if err = comm.AddClient(cPtr); err != nil {
        return errors.New(fmt.Sprintf(
            "(s.AddClient) %v", err))
    }

    cPtr.SetCAChans(s.caChan, comm.caChan)

    return
}

func (s *Server) shouldCreateComm(commID string) bool {
    // TODO: properly check whether or not this Server w/ ID s.ID
    //       should actually create a Community w/ ID CommID
    return true
}

func (s *Server) RemoveClient(cPtr *Client) (err error) {
    comm, ok := s.Comms[cPtr.CommID]
    if !ok {
        return errors.New(fmt.Sprintf(
            "(s.RemoveClient) Comm %s DNE", cPtr.CommID))
    }

    // remove Client from current Community
    if err = comm.RemoveClient(cPtr); err != nil {
        return errors.New(fmt.Sprintf(
            "(s.RemoveClient) %v", err))
    }

    cPtr.RemoveCAChans()

    // replace Client in "root" Community
    cPtr.CommID = ROOT_COMM_ID
    s.AddClient(cPtr)

    return
}