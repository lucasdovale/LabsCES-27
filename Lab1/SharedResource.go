package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}

func doServerJobCS(ServConn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, _, err := ServConn.ReadFromUDP(buf) // trocar primeiro _ por n
		msg := string(buf[0:n])
		mensagens := strings.Split(msg, "/")
		idReceived := mensagens[0]
		clock := mensagens[1]
		mensagem := mensagens[2]
		fmt.Printf("Recebi a mensagem '%s' do processo %s com logicalClock %s\n", mensagem, idReceived, clock)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		time.Sleep(time.Second * 1)
	}
}

func main() {
	Address, err := net.ResolveUDPAddr("udp", ":10001")
	CheckError(err)
	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)
	defer Connection.Close()
	for {
		go doServerJobCS(Connection)
		time.Sleep(time.Second * 1)
	}
}
