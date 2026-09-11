package main

import (
    "fmt"
    "net"
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
    
    for {
        resp := NewResp(conn)
        val, err := resp.Read()

        if err != nil {
            fmt.Println(err)
            return
        }

        fmt.Println(val)

        // buf := make([]byte, 1024)

        // // read message from client
        // _, err = conn.Read(buf)
        // if err != nil {
        //     if err == io.EOF {
        //         break
        //     }
        //     fmt.Println("Error reading from client: ", err.Error())
        //     os.Exit(1)
        // }

        conn.Write([]byte("+OK\r\n"))
    }            
}

