package main

import (
	"fmt"
	"io"
	"net"
	"strings"
)

func main() {
	fmt.Println("Listening on port: 6379")

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	resp := NewResp(conn)
	writer := NewWriter(conn)

	for {
		val, err := resp.read()

		if err != nil {
			if err == io.EOF {
				fmt.Println("Client disconnected.")
				return
			}

			fmt.Println(err)
			return
		}

		if val.typ != "array" {
			fmt.Println("Expected array value")
			continue
		}

		if len(val.array) == 0 {
			fmt.Println("Expected array length > 0")
			continue
		}

		command := strings.ToUpper(val.array[0].bulk)
		args := val.array[1:]

		handler, ok := Handlers[command]

		if !ok {
			fmt.Println("Invalid command")

			err := writer.write(Value{
				typ: "error",
				str: "ERR unknown command",
			})

			if err != nil {
				fmt.Println("Write error:", err)
				return
			}

			continue
		}

		result := handler(args)

		if err := writer.write(result); err != nil {
			fmt.Println("Write error:", err)
			return
		}
	}
}
