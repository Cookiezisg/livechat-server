package main

import (
	"fmt"
	"net"
	"sync"
)

type User struct {
	name string
	id   string
	con  net.Conn
	msg  chan string
}

var userList = make(map[string]User)
var userMutex sync.Mutex
var broadcastChan = make(chan string, 10)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server listening on port 8080")

	go broadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}
		defer conn.Close()
		fmt.Println("Client connected:", conn.RemoteAddr())

		go handler(conn)
	}
}

func handler(conn net.Conn) {

	defer func() {
		fmt.Println("Client disconnected:", conn.RemoteAddr())
		conn.Close()
		userMutex.Lock()
		delete(userList, conn.RemoteAddr().String())
		userMutex.Unlock()
	}()

	fmt.Fprintf(conn, "Welcome to the live chat server!\n")

	newUser := User{
		id:   conn.RemoteAddr().String(),
		name: conn.RemoteAddr().String(),
		con:  conn,
		msg:  make(chan string, 10),
	}

	userMutex.Lock()
	userList[newUser.id] = newUser
	userMutex.Unlock()

	fmt.Printf("New user connected: %s\n", newUser.name)
	broadcastChan <- fmt.Sprintf("%s has joined the chat. \n", newUser.name)

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			return
		}
		message := string(buf[:n])
		fmt.Printf("Received message: %s", message)

		if message == "who\n" {
			userMutex.Lock()
			var userNames string
			for _, user := range userList {
				userNames += user.name + "\n"
			}
			userMutex.Unlock()
			conn.Write([]byte("Current users:\n" + userNames))
			continue
		} else if message == "exit\n" {
			broadcastChan <- fmt.Sprintf("%s has left the chat.\n", newUser.name)
			return
		} else if message == "rename\n" {
			conn.Write([]byte("Enter new name: "))
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Error reading new name:", err)
				return
			}
			newName := string(buf[:n-1]) // Remove newline character
			userMutex.Lock()
			newUser.name = newName
			userList[newUser.id] = newUser
			userMutex.Unlock()
			conn.Write([]byte(fmt.Sprintf("Name changed to %s\n", newName)))
		} else if message == "help\n" {
			helpMessage := "Available commands:\n" +
				"who - List all connected users\n" +
				"rename - Change your display name\n" +
				"exit - Leave the chat\n" +
				"help - Show this help message\n"
			conn.Write([]byte(helpMessage))
		} else {
			broadcastChan <- fmt.Sprintf("%s: %s", newUser.name, message)
		}
	}
}

func broadcaster() {
	for {
		msg := <-broadcastChan
		userMutex.Lock()
		for _, user := range userList {
			go func(u User, m string) {
				_, err := u.con.Write([]byte(m))
				if err != nil {
					fmt.Println("Error broadcasting to user:", err)
				}
			}(user, msg)
		}
		userMutex.Unlock()
	}
}
