package main

import (
	"io"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
)

// Global Client UserID value
const INVALID_CLIENT_USERID = 0 // id 0 is invalid
var CLIENT_USERID_MUTEX sync.Mutex
var CLIENT_USERID uint32 = INVALID_CLIENT_USERID

// BLOCKING: CLIENT_USERID_MUTEX
func NextClientID() (uint32) {
	CLIENT_USERID_MUTEX.Lock()
	defer CLIENT_USERID_MUTEX.Unlock()

	CLIENT_USERID++
	if (CLIENT_USERID == INVALID_CLIENT_USERID) {
		CLIENT_USERID++
	}

	return CLIENT_USERID
}

type Client struct {
	ID			uint32 // TODO: use user deviceID hash instead
	Name		string // FB first name
	ServerID	string // region name
	CommID		string // current neighbourhood

	conn 		net.Conn
	connReader	*bufio.Reader

	authComplete	chan bool

	serverCAChan	chan *ClientAction
	commCAChan 		chan *ClientAction
	caChanRWMutex	sync.RWMutex

	disconnected	bool
}

// will close conn if err != nil
func NewClient(connPtr *net.Conn) (c *Client, err error) {
	defer func(connPtr *net.Conn, err *error) {
		if *err != nil {
			log.Println("DEBUG: NewClient() failed; closing conn")
            (*connPtr).Close() // ignoring errors
		}
	}(connPtr, &err)

	c = new(Client)
	c.conn = *connPtr
	c.connReader = bufio.NewReader(c.conn)
	c.disconnected = false

	c.authComplete = make(chan bool)
	c.RemoveCAChans()
	

	if err = c.requestAuth(); err != nil {
		return nil, err
	}

	sID, err := ServerIDFromIP(c.conn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	c.ServerID = sID

	go c.readLoop()

	return
}

func (c *Client) ToString() string {
	return fmt.Sprintf("Client %s (%v)", c.Name, c.ID)
}

func (c *Client) RemoveCAChans() {
	// TODO: import "https://github.com/matryer/resync"
	/*
	c.removeCAOnce.Do() {
		c.caChanRWMutex.Lock()
		c.serverCAChan = nil
		c.commCAChan = nil
	}
	*/
}

func (c *Client) SetCAChans(sCAChan chan *ClientAction,
	commCAChan chan *ClientAction) {
	// TODO: import "https://github.com/matryer/resync"
	/*
	c.removeCAOnce.Do() {
		c.serverCAChan = sCAChan
		c.commCAChan = commCAChan
		c.caChanRWMutex.Unlock()
	}
	*/
}

func (c *Client) Disconnect() {
	// TODO: atomic boolean
	if c.disconnected {
		log.Printf("%s already disconnected.\n", c.ToString())
		return
	}
	c.disconnected = true
	log.Printf("Disconnecting %s (id: %v)\n", c.Name, c.ID)
	c.conn.Close() // ignoring errors

	// TODO: tell Community/Server/ServerWrapper to remove Client
}

// Retreives values for c.ID & c.Name from c.conn
func (c *Client) requestAuth() (err error) {
	defer log.Println("Client.requestAuth() success.")

	/* TODO:
	c.WriteMsg(MsgClientAuth)
	<-c.authComplete // close once auth has completed
	*/

	// TODO: c.ID := c.ReadMsg() // MsgClientID
	c.ID = NextClientID()
	// TODO: c.Name := c.ReadMsg() // MsgClientName
	c.Name = fmt.Sprintf("Client_%v", c.ID)

	return
}

func (c *Client) readLoop() {
	for {
		msg, err := c.readMsg()
		if err != nil {
			log.Printf("Failed to read message for %s.\n", c.ToString())
			// if err == io.EOF {
				c.Disconnect()
				break
			// } else {
		}

		log.Printf("Read message of type %s from %s.\n",
			msg.TypeToString(), c.ToString())

		// TODO: c.handleMsg(msg)
		// ... close(c.authComplete) // indicate auth has completed
		// c.caChanRWMutex.RLock()
		// if msg.IsCA() { c.serverCAChan <- msg.ToCA() }
		// c.caChanRWMutex.RUnlock()
	}

	log.Printf("Exiting readLoop for %s.\n", c.ToString())
}

func (c *Client) readMsg() (msg *Message, err error) {
	var (
		msgType 	uint8
		msgLen 		uint32
	)

	// Read 'headers' of the message
	err = binary.Read(c.connReader, binary.BigEndian, &msgType)
	if err != nil {
		return nil, err
	}
	err = binary.Read(c.connReader, binary.BigEndian, &msgLen)
	if err != nil {
		return nil, err
	}

	msgData := make([]byte, msgLen)
	// read from c.conn until msgData is full (read msgLen bytes)
	_, err = io.ReadFull(c.connReader, msgData)
	if err != nil {
		return nil, err
	}

	msg, err = MsgFromBinary(msgType, msgData)
	if err != nil {
		return nil, err
	}

	return // msg, err
}

/*func (c *Client) SendServerCA(caPtr *ClientAction) {

}

func (c *Client) SendCommCA(caPtr *ClientAction) {

}*/

func (c *Client) WriteMsg(msg *Message) (err error) {
	dataBin, err := msg.ToBinary()
	if err !=  nil {
		return err
	}

	buf := new(bytes.Buffer)

	// Write message 'headers'
	err = binary.Write(buf, binary.BigEndian, msg.Type)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, uint32(len(dataBin)))
	if err != nil {
		return err
	}

	// attempt to write msg.Data's binary encoding
	_, err = buf.Write(dataBin)
	if err != nil {
		return err
	}

	// attempt to write full message into c.conn
	_, err = c.conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return
}