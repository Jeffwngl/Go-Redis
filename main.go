package main

import (
	"fmt"
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

	conn, err := l.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	resp := NewResp(conn)
	writer := NewWriter(conn)

	for {
		val, err := resp.read()

		if err != nil {
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
			writer.write(Value{typ: "string", str: ""})
			continue
		}

		// fmt.Println(val)

		result := handler(args)
		writer.write(result)
	}
}
