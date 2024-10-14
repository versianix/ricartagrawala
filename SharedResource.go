package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Message struct {
	Type  int
	Clock int
	From  int
	Text  string
}

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func main() {
	Address, err := net.ResolveUDPAddr("udp", ":10000")
	CheckError(err)
	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)
	defer Connection.Close()

	buf := make([]byte, 1024)
	for {
		n, _, err := Connection.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			continue
		}
		fmt.Printf("Received message from Process %d (Clock: %d): %s\n", msg.From, msg.Clock, msg.Text)
	}
}
