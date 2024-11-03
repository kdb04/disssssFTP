package main

import (
    "fmt"
    "net"
    "os"
)

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println("Error starting server:", err)
        os.Exit(1)
    }
    defer ln.Close()

    fmt.Println("Server is listening on port 8080...")

    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println("Error accepting connection:", err)
            continue         }

        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    buf := make([]byte, 1024)
    n, err := conn.Read(buf)
    if err != nil {
        fmt.Println("Error reading from connection:", err)
        return
    }

    fmt.Printf("Received: %s\n", buf[:n])
}
