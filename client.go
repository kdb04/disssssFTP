// test_clients.go
package main

import (
    "fmt"
    "math/rand"
    "net"
    "sync"
    "time"
)

func main() {
    var wg sync.WaitGroup
    numClients := 10 

    for i := 0; i < numClients; i++ {
        wg.Add(1)
        go func(clientID int) {
            defer wg.Done()

            conn, err := net.Dial("tcp", "localhost:8080")
            if err != nil {
                fmt.Println("Error connecting to server:", err)
                return
            }
            defer conn.Close() 

            time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

            message := fmt.Sprintf("Hello from client %d!", clientID)
            _, err = conn.Write([]byte(message))
            if err != nil {
                fmt.Println("Error sending message:", err)
                return
            }

            fmt.Printf("Client %d: Message sent: %s\n", clientID, message)
        }(i)
    }

    wg.Wait()
}
