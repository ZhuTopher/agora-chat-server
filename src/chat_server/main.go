package main

import (
    "bufio"
    "errors"
    "fmt"
    "log"
    "net"
    "os"
    "strings"
)

// OS Env variable fetching functions

// Return the deployed service's HOST_IP
func getHostIP() (string, error) {
    // TODO: different IPS for region-based cloud servers

    host := os.Getenv("HOST_IP")
    if host == "" {
        return "", errors.New("Missing/empty host ip")
    }
    return host, nil
}

// Return the deployed service's TCP_PORT
func getTCPPort() (string, error) {
    host := os.Getenv("TCP_PORT")
    if host == "" {
        return "", errors.New("Missing/empty TCP port")
    }
    return host, nil
}

// Return the deployed service's API_IP
func getAPIPort() (string, error) {
    host := os.Getenv("API_PORT")
    if host == "" {
        return "", errors.New("Missing/empty API port")
    }
    return host, nil
}

func ServerIDFromIP(ipAddr string) (serverID string, err error) {
    // TODO: determine ServerID based on c.conn IP addr

    return "main", nil
}

// ctor private, accessible to main only
func newServerWrapper() (sw *ServerWrapper, err error) {
    sw = new(ServerWrapper)
    sw.Servers = make(map[string]*Server)
    sw.done = make(chan bool)

    hostName, err := getHostIP()
    if (err != nil) {
        return nil, errors.New(fmt.Sprintf(
            "Unable to fetch server's public ip: %v", err))
    }
    tcpPort, err := getTCPPort()
    if (err != nil) {
        return nil, errors.New(fmt.Sprintf(
            "Unable to fetch server's tcp port: %v", err))
    }
    serverAddr, err := net.ResolveTCPAddr("tcp", hostName + ":" + tcpPort)
    if (err != nil) {
        return nil, errors.New(fmt.Sprintf(
            "Unable to resolve server's tcp address: %v", err))
    }

    // setup listener for incoming TCP connections
    // net.ListenTCP("tcp", )
    sw.tcpl, err = net.ListenTCP("tcp", serverAddr)
    if err != nil {
        return nil, errors.New(fmt.Sprintf(
            "Unable to listen on TCP addr %s: %v",
            serverAddr, err))
    }
    // err = server.tcpl.SetTimeout(1e9) ...
    sw.connChan = make(chan *net.Conn)
    sw.caChan = make(chan *ClientAction)

    sw.running = true

    return // sw, nil
}

func main() {
    sw, err := newServerWrapper()
    if err != nil {
        log.Fatalf("Failed to create ServerWrapper: %v\n", err)
    }
    defer func() {
        err := sw.Shutdown()
        if err != nil {
            log.Printf("Error shutting down servers: %v\n", err)
        }
    }()

    sw.loopWG.Add(3)
    go sw.acceptLoop()
    go sw.clientBuilderLoop()
    go sw.controlLoop()

    stdinChan := make(chan string)
    go func(stdinChan chan string) {
        reader := bufio.NewReader(os.Stdin)
        for {
            text, err := reader.ReadString('\n')
            if err != nil {
                return // io.EOF
            }
            if (strings.Compare(text, "q\n")) == 0 {
                stdinChan <- text
            }
        }
    }(stdinChan)
    select {
    case <-stdinChan:
        log.Println("MANUAL SERVER TERMINATION INPUTTED")
        // TODO: listen for manual shutdown (keystroke?)
    }
}
