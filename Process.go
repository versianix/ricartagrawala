package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Message types
const (
	REQUEST = iota
	REPLY
)

// States for the process
const (
	RELEASED = iota
	WANTED
	HELD
)

// Message struct to be sent between processes
type Message struct {
	Type  int
	Clock int
	From  int
	Text  string
}

var (
	myID               int                  // Process ID
	myPort             string               // My server port
	nServers           int                  // Total number of processes
	CliConn            map[int]*net.UDPConn // Map from process IDs to connections
	ServConn           *net.UDPConn         // My server connection
	err                string
	clock              int        // Logical clock
	T                  int        // Timestamp for request
	state              int        // State: RELEASED, WANTED, or HELD
	replyCount         int        // Number of replies received
	mutex              sync.Mutex // Mutex for shared variables
	replyChan          chan bool  // Channel to signal when all replies are received
	queue              []Message  // Queue of deferred requests
	sharedResourceAddr string     // Address of the shared resource
)

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func PrintError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
	}
}

func readInput(ch chan string) {
	// listens for keyboard input
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}

func doServerJob() {
	buf := make([]byte, 1024)
	// handles incoming messages
	for {
		n, addr, err := ServConn.ReadFromUDP(buf)
		PrintError(err)
		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		PrintError(err)

		// process message
		mutex.Lock()
		updateClockOnReceive(msg.Clock)
		switch msg.Type {
		case REQUEST:
			handleRequest(msg, addr)
		case REPLY:
			handleReply(msg)
		}
		mutex.Unlock()
	}
}

func handleRequest(msg Message, addr *net.UDPAddr) {
	fmt.Println("Received CS REQUEST from process", msg.From)
	if state == HELD || (state == WANTED && compareTimestamp(msg, myID, clock)) {
		queue = append(queue, msg)
	} else {
		sendReply(msg.From)
	}
}

func handleReply(msg Message) {
	replyCount++
	fmt.Println("Received REPLY from process", msg.From, "(", replyCount, "replies received of", nServers-1, ")")
	if replyCount == nServers-1 {
		replyChan <- true
	}
}

func compareTimestamp(msg Message, myID int, myClock int) bool {
	if T < msg.Clock {
		return true
	} else if msg.Clock == myClock {
		return myID < msg.From
	}
	return false
}

func sendReply(destID int) {
	// clock isn't updated
	msg := Message{
		Type:  REPLY,
		Clock: clock,
		From:  myID,
		Text:  "Reply sent from " + fmt.Sprint(myID),
	}
	jsonMsg, err := json.Marshal(msg)
	PrintError(err)
	conn, ok := CliConn[destID]
	if !ok {
		fmt.Println("Connection to process", destID, "not found")
		return
	}
	_, err = conn.Write(jsonMsg)
	PrintError(err)
	fmt.Println("Sent REPLY to process", destID)
}

func doClientJob() {
	// initiates a REQUEST to enter the critical section
	mutex.Lock()
	if state == RELEASED {
		state = WANTED
		fmt.Println("Current state: WANTED")
		time.Sleep(2 * time.Second)
		clock++
		T = clock
		replyCount = 0
		mutex.Unlock()

		sendRequestToAll(T)

		// Wait until all replies are received
		<-replyChan

		mutex.Lock()
		state = HELD
		fmt.Println("Current state: HELD")
		mutex.Unlock()

		enterCriticalSection()

		mutex.Lock()
		state = RELEASED
		fmt.Println("Current state: RELEASED")
		replyDeferredRequests()
		mutex.Unlock()
	} else {
		fmt.Println("Request to enter CS ignored; already in WANTED or HELD state.")
		mutex.Unlock()
	}
}

func sendRequestToAll(T int) {
	for _, conn := range CliConn {
		msg := Message{
			Type:  REQUEST,
			Clock: T,
			From:  myID,
			Text:  "Request to enter CS from " + fmt.Sprint(myID),
		}
		jsonMsg, err := json.Marshal(msg)
		PrintError(err)
		_, err = conn.Write(jsonMsg)
		PrintError(err)
	}
	fmt.Println("Sent CS REQUEST to all processes")
}

func enterCriticalSection() {
	fmt.Println("Entered Critical Section")
	sendMessageToSharedResource()
	// sleep to simulate critical section
	time.Sleep(5 * time.Second)
	fmt.Println("Exited Critical Section")
}

func sendMessageToSharedResource() {
	ServerAddr, err := net.ResolveUDPAddr("udp", sharedResourceAddr)
	CheckError(err)
	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	CheckError(err)
	defer Conn.Close()

	mutex.Lock()
	msg := Message{
		Type:  -1, // Type -1 indicates message to shared resource
		Clock: clock,
		From:  myID,
		Text:  "Entering the Critical Section",
	}
	mutex.Unlock()

	jsonMsg, err := json.Marshal(msg)
	PrintError(err)
	_, err = Conn.Write(jsonMsg)
	PrintError(err)
}

func replyDeferredRequests() {
	for _, msg := range queue {
		sendReply(msg.From)
	}
	queue = []Message{}
}

func updateClockOnReceive(receivedClock int) {
	if clock < receivedClock {
		clock = receivedClock
	}
	clock++
}

func initConnections() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run Process.go <ProcessID> <Address1> <Address2> ... <SharedResourceAddress>")
		os.Exit(1)
	}
	myID = atoi(os.Args[1])
	addresses := os.Args[2 : len(os.Args)-1]     // All addresses except the last one
	sharedResourceAddr = os.Args[len(os.Args)-1] // Last argument is the shared resource address
	nServers = len(addresses)
	CliConn = make(map[int]*net.UDPConn)

	myAddress := addresses[myID-1]
	myPort = ":" + strings.Split(myAddress, ":")[1] // Extract port from myAddress

	// Setup server connection
	ServerAddr, err := net.ResolveUDPAddr("udp", myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)

	// Build connections to other processes
	for i, addr := range addresses {
		pid := i + 1 // Process IDs start from 1
		if pid == myID {
			continue // Skip our own process
		}
		ServerAddr, err := net.ResolveUDPAddr("udp", addr)
		CheckError(err)
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		CheckError(err)
		CliConn[pid] = Conn
	}
}

func atoi(s string) int {
	// converts string to integer
	n, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println("Invalid integer:", s)
		os.Exit(1)
	}
	return n
}

func main() {
	// Initialization
	initConnections()
	// Defer closing connections
	defer ServConn.Close()
	for _, conn := range CliConn {
		defer conn.Close()
	}
	clock = 0
	state = RELEASED
	fmt.Println("Current state: RELEASED")
	replyChan = make(chan bool)
	queue = []Message{}

	ch := make(chan string)
	go readInput(ch)
	go doServerJob()

	for {
		select {
		case x, valid := <-ch:
			if valid {
				if x == "x" {
					go doClientJob()
				} else if x == fmt.Sprint(myID) {
					mutex.Lock()
					clock++
					fmt.Println("Internal action; clock incremented to", clock)
					mutex.Unlock()
				} else {
					fmt.Println("Invalid input")
				}
			} else {
				fmt.Println("Closed channel!")
			}
		default:
			time.Sleep(time.Second * 1)
		}
		time.Sleep(time.Second * 1)
	}
}
