package main

import (
	"log"
)

// Client action (eg: server-join, community-leave, etc.) wrapper
type ClientAction struct {
	ClientID 	uint32
	Action     	interface{}
}

type JoinServer struct {
	ServerID	string
	ClientPtr	*Client
}

type JoinComm struct {
	CommID	string
	ClientPtr	*Client
}

// requires caPtr.Action points to a JoinServer
func (sw *ServerWrapper) CAJoinServer(caPtr *ClientAction) {
	js := caPtr.Action.(JoinServer)
	sID := js.ServerID
	log.Printf("(sw) Moving Client %v (%v) to Server %v\n",
		js.ClientPtr.Name, js.ClientPtr.ID, sID)

	toServer, ok := sw.Servers[sID]
	if !ok {
		toServer = NewServer(sID)
		sw.Servers[sID] = toServer
	}

	cPtr := js.ClientPtr
	cServerID := cPtr.ServerID
	if fromServer, ok := sw.Servers[cServerID]; ok {
		fromServer.RemoveClient(cPtr) // ignore errors
	}

	toServer.caChan <- caPtr
}

// requires caPtr.Action points to a JoinServer
func (s *Server) CAJoinServer(caPtr *ClientAction) {
	js := caPtr.Action.(JoinServer)
	cPtr := js.ClientPtr

	if err := s.AddClientToRootComm(cPtr); err != nil {
		log.Printf("Unable to add %s to Server %s:\n%v\n",
			cPtr.ToString(), s.ID, err)
	}
}