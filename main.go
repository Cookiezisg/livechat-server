package main

import (
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server listening on port 8080")

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
	fmt.Fprintf(conn, "Welcome to the live chat server!\n")
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			return
		}
		message := string(buf[:n])
		fmt.Printf("Recei ved message: %s", message)
		fmt.Fprintf(conn, "You said: %s", message)
	}
}
