package main

import (
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "os"
    //"path/filepath"
)

const (
    serverAddress = "localhost:17000"
    bufferSize    = 4096
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Please provide the filename")
        return
    }
    fileName := os.Args[1]
    //filePath := filepath.Join("client", fileName) 

    if _, err := os.Stat(fileName); os.IsNotExist(err){
        fmt.Println("Error: File doesn't exist.")
        return 
    }

    // Connecting to server
    conn, err := net.Dial("tcp", serverAddress)
    if err != nil {
        fmt.Println("Error connecting to server:", err)
        return
    }
    defer conn.Close()

    //Sending file to server
    fileNameBytes := []byte(fileName)
    fileNameLen := int32(len(fileNameBytes))
    err = binary.Write(conn, binary.LittleEndian, fileNameLen)
    if err != nil {
        fmt.Println("Error sending filename length:", err)
        return
    }

    _, err = conn.Write(fileNameBytes)
    if err != nil {
        fmt.Println("Error sending filename:", err)
        return
    }

    file, err := os.Open(fileName)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // Sending file content
    buf := make([]byte, bufferSize)
    for {
        n, err := file.Read(buf)
        if err != nil {
            if err == io.EOF {
                fmt.Println("File sent successfully:", fileName)
                break
            } else {
                fmt.Println("Error reading file:", err)
                return
            }
        }
        if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
            fmt.Println("Error sending file content:", writeErr)
            return
        }
    }

    confirmation := make([]byte, 4)
    _, err = conn.Read(confirmation)
    if err == nil && string(confirmation) == "Done" {
        fmt.Println("Server confirmed successful file transfer.")
    } else {
        fmt.Println("Error receiving server confirmation:", err)
    }
}
