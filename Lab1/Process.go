package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var idInt int
var logicalClock int

var WANTED bool
var HELD bool
var RELEASED bool

var replys int
var queue []int

var err string
var myPort string
var nServers int
var CliConn []*net.UDPConn
var ServConn *net.UDPConn

func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}

func PrintError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
	}
}

func readInput(ch chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}

func initConnections() {
	logicalClock = 0

	WANTED = false
	HELD = false
	RELEASED = true

	id := os.Args[1]
	idInt, _ = strconv.Atoi(id)
	myPort = os.Args[idInt+1]
	nServers = len(os.Args) - 2

	CliConn = make([]*net.UDPConn, nServers)

	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)

	for servidores := 0; servidores < nServers; servidores++ {
		if servidores != idInt-1 {
			ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+os.Args[2+servidores])
			CheckError(err)
			Conn, err := net.DialUDP("udp", nil, ServerAddr)
			CliConn[servidores] = Conn
			CheckError(err)
		}
	}
	nServers--
}

func verificaReplys(replys int) bool {
	if replys == nServers {
		return true
	} else {
		return false
	}
}

func atualizaClock(otherLogicalClock int) {
	if otherLogicalClock >= logicalClock {
		logicalClock = otherLogicalClock + 1
	} else {
		logicalClock++
	}
}

func entrarNaCS(mutex *sync.Mutex) {
	fmt.Println("\nEntrei na CS")

	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	CheckError(err)
	Conn, err := net.DialUDP("udp", nil, ServerAddr)

	mutex.Lock()
	id := strconv.Itoa(idInt)
	clockMsg := strconv.Itoa(logicalClock)
	mutex.Unlock()

	txtToCS := "Consegui entrar na CS :D"
	msg := id + "/" + clockMsg + "/" + txtToCS
	bufToCS := []byte(msg)
	_, err = Conn.Write(bufToCS)
	if err != nil {
		fmt.Println(msg, err)
	}
	time.Sleep(time.Second * 20)

	// AO FAZER USO DA CS, MUDA ESTADO PARA RELEASED
	mutex.Lock()
	HELD = false
	RELEASED = true
	mutex.Unlock()

	// ENVIA REPLY PARA TODOS DA FILA
	mensagem := "REPLY"
	mutex.Lock()
	msg = id + "/" + clockMsg + "/" + mensagem
	mutex.Unlock()
	buf := []byte(msg)

	mutex.Lock()
	for _, i := range queue {
		_, err := CliConn[i-1].Write(buf)
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}
	mutex.Unlock()

	fmt.Println("Saí da CS")

	mutex.Lock()
	fmt.Println("Enviando reply para ", queue)
	queue = nil
	replys = 0
	mutex.Unlock()
}

func doServerJob(mutex *sync.Mutex) {
	buf := make([]byte, 1024)

	for {
		n, _, err := ServConn.ReadFromUDP(buf)
		msg := string(buf[0:n])
		mensagens := strings.Split(msg, "/")
		idReceived := mensagens[0]
		clock := mensagens[1]
		mensagem := mensagens[2]
		fmt.Println("Recebi", mensagem, "do processo", idReceived, "com logicalClock", clock)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		time.Sleep(time.Second * 1)

		idReceivedInt, _ := strconv.Atoi(idReceived)
		otherLogicalClock, _ := strconv.Atoi(string(buf[0:n]))

		if mensagem == "REQUEST" {
			// ATUALIZA CLOCK
			mutex.Lock()
			atualizaClock(otherLogicalClock)
			mutex.Unlock()
			fmt.Printf("Atualizando logicalClock: %d\n", logicalClock)

			if err != nil {
				fmt.Println("Error: ", err)
			}
			if HELD || (WANTED && logicalClock < otherLogicalClock) || (WANTED && logicalClock == otherLogicalClock && idInt < idReceivedInt) {
				// ENFILEIRA
				fmt.Println("Enfileirando o processo", idReceived, "\n")
				mutex.Lock()
				queue = append(queue, idReceivedInt)
				mutex.Unlock()
			} else {
				// ENVIA REPLY
				fmt.Println("Envio REPLY para o processo", idReceived, "\n")
				mutex.Lock()
				id := strconv.Itoa(idInt)
				clockMsg := strconv.Itoa(logicalClock)
				mutex.Unlock()
				mensagem := "REPLY"
				msg := id + "/" + clockMsg + "/" + mensagem
				buf := []byte(msg)
				_, err := CliConn[idReceivedInt-1].Write(buf)
				if err != nil {
					fmt.Println("Error: ", err)
				}
			}
		} else if mensagem == "REPLY" {
			// ATUALIZA CLOCK
			mutex.Lock()
			atualizaClock(otherLogicalClock)
			mutex.Unlock()
			fmt.Printf("Atualizando logicalClock: %d\n", logicalClock)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			// INCREMENTA REPLYS
			mutex.Lock()
			replys++
			mutex.Unlock()
			fmt.Printf("Incrementando replys: %d\n", replys)
		}
		if verificaReplys(replys) && WANTED {
			// AO RECEBER TODOS REPLYS, MUDA ESTADO PARA HELD E ENTRA NA CS
			mutex.Lock()
			WANTED = false
			HELD = true
			go entrarNaCS(mutex)
			mutex.Unlock()
		}

	}
}

func doClientJob(otherProcess int, clock int, mensagem string, mutex *sync.Mutex) {
	mutex.Lock()
	id := strconv.Itoa(idInt)
	mutex.Unlock()
	clockMsg := strconv.Itoa(clock)
	msg := id + "/" + clockMsg + "/" + mensagem
	buf := []byte(msg)
	if mensagem == "REQUEST" {
		mutex.Lock()
		_, err := CliConn[otherProcess].Write(buf)
		mutex.Unlock()
		if err != nil {
			fmt.Println(msg, err)
		}
	}
	time.Sleep(time.Second * 1)
}

func main() {
	var mutex sync.Mutex
	initConnections()
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		if i != idInt-1 {
			defer CliConn[i].Close()
		}
	}

	fmt.Printf("ID: %d\nPorta: %s\nlogicalClock: %d\n\n", idInt, myPort, logicalClock)

	ch := make(chan string)
	go readInput(ch)

	mutex.Lock()
	go doServerJob(&mutex)
	mutex.Unlock()

	for {
		select {
		case idOtherProcess, valid := <-ch:
			if valid {
				if idOtherProcess == "x" && RELEASED {
					RELEASED = false
					WANTED = true
					logicalClock++
					fmt.Printf("\nAtualizando logicalClock: %d \n", logicalClock)
					fmt.Printf("Processo %d quer acessar a CS", idInt)
					fmt.Println("\n")
					for i := 1; i <= nServers+1; i++ {
						if i != idInt {
							fmt.Printf("Envio REQUEST para o processo %d \n", i)
							mutex.Lock()
							go doClientJob(i-1, logicalClock, "REQUEST", &mutex)
							mutex.Unlock()
						}
					}
				} else if idOtherProcess == "x" && !RELEASED {
					fmt.Println("Entrada ignorada!")
				} else {
					idOtherProcessInt, err := strconv.Atoi(idOtherProcess)
					CheckError(err)
					if idOtherProcessInt >= 1 && idOtherProcessInt <= nServers+1 {
						if idOtherProcessInt != idInt {
							fmt.Printf("Enviar mensagem para o processo %s \n", idOtherProcess)
							fmt.Printf("Atualizando logicalClock: %d \n", logicalClock)
							mutex.Lock()
							go doClientJob(idOtherProcessInt-1, logicalClock, "REQUEST", &mutex)
							mutex.Unlock()
						} else {
							logicalClock++
							fmt.Printf("Atualizando logicalClock: %d \n", logicalClock)
						}
					}
				}
			} else {
				fmt.Println("Canal fechado!")
			}
		default:
			time.Sleep(time.Second * 1)
		}
		time.Sleep(time.Second * 1)
	}
}
