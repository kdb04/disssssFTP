package main

import (
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "os"
    "path/filepath"
)

const (
    serverPort = "17000"
    bufferSize = 4096
    uploadDir  = "uploads"
)

func main() {
    //Making upload directory
    os.MkdirAll(uploadDir, os.ModePerm)

    // Starting server
    ln, err := net.Listen("tcp", ":"+serverPort)
    if err != nil {
        fmt.Println("Error starting server:", err)
        return
    }
    defer ln.Close()
    fmt.Println("Server is listening on port", serverPort)

    //Handling multiple clients concurrently
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println("Error accepting connection:", err)
            continue
        }
        go handleConnection(conn) 
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    var fileNameLen int32
    err := binary.Read(conn, binary.LittleEndian, &fileNameLen)
    if err != nil {
        fmt.Println("Error reading filename length:", err)
        return
    }

    fileNameBuf := make([]byte, fileNameLen)
    _, err = conn.Read(fileNameBuf)
    if err != nil {
        fmt.Println("Error reading filename:", err)
        return
    }
    fileName := string(fileNameBuf)
    filePath := filepath.Join(uploadDir, fileName)

    fmt.Println("Receiving file:", fileName)

    //Creating file in upload directory
    file, err := os.Create(filePath)
    if err != nil {
        fmt.Println("Error creating file:", err)
        return
    }
    defer file.Close()

    buf := make([]byte, bufferSize)
    for {
        n, err := conn.Read(buf)
        if err != nil {
            if err == io.EOF {
                fmt.Println("File received successfully:", fileName)
                conn.Write([]byte("Done")) 
            } else {
                fmt.Println("Error receiving file:", err)
            }
            break
        }

        // Write and display contents of the file
        if _, err := file.Write(buf[:n]); err != nil {
            fmt.Println("Error writing to file:", err)
            return
        }

        fmt.Print(string(buf[:n]))
    }
}
