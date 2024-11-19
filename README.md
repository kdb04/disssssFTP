# Distributed File Orchestration and Synchronization: Multi-Node Data-Transfer-Framework for Linux

## Introduction

This system facilitates secure file transfer operations between clients and a server over TCP. It supports authentication, multiple file operations, and lays the groundwork for future enhancements like encryption.

## System Components

### Client (`client.go`)

- **Purpose**: Provides a command-line interface for users to interact with the server for file operations.
- **Key Functionalities**:
  - **Authentication**: Users authenticate with a username and password.
  - **File Operations**:
    - Upload files to the server.
    - Download files from the server.
    - View file contents.
    - Delete files on the server.
    - List files stored on the server.

### Server (`server.go`)

- **Purpose**: Handles client connections, authentication, and executes file operations requested by authenticated clients.
- **Key Functionalities**:
  - **Authentication**: Verifies user credentials against a stored credentials file (`id_passwd.txt`).
  - **Session Management**: Tracks authenticated sessions and ensures idle connections are terminated after a timeout.
  - **File Operations**: Processes file operation requests from clients within their designated directories.

## Protocol Specifications

### Connection and Authentication

1. **Establish TCP Connection**:
   - Client connects to server on port `9090`.
2. **Send Credentials**:
   - Client sends 

username:password

 followed by a newline character.
3. **Server Response**:
   - On success: `Authentication successful. You are now connected.\n`
   - On failure: Appropriate error message and termination of the connection.

### File Operations

After successful authentication, the client can perform the following operations by sending an operation code followed by the required data.

#### Operation Codes

- `1`: Upload File
- `2`: Download File
- `3`: View File
- `4`: Delete File
- `5`: List Files

#### Upload File (Operation Code `1`)

1. **Client**:
   - Sends operation code `1`.
   - Sends filename length (int32) and filename.
   - Sends file size (int64).
   - Sends file content in byte chunks.
2. **Server**:
   - Receives file metadata and content.
   - Stores the file in the user's directory.
   - Sends acknowledgment `Done\n` upon completion.

#### Download File (Operation Code `2`)

1. **Client**:
   - Sends operation code `2`.
   - Sends filename length (int32) and filename.
2. **Server**:
   - Checks if the file exists.
     - If it does, sends file size (int64) and then file content.
     - If not, sends file size `0` and an error message.

#### View File (Operation Code `3`)

1. **Client**:
   - Sends operation code `3`.
   - Sends filename length (int32) and filename.
2. **Server**:
   - Checks if the file exists.
     - If it does, sends status `1`, file size, and file content.
     - If not, sends status `0`.
3. **Client**:
   - Displays the file content to the user.

#### Delete File (Operation Code `4`)

1. **Client**:
   - Sends operation code `4`.
   - Sends filename length (int32) and filename.
2. **Server**:
   - Attempts to delete the specified file.
   - Sends status `1` on success or `0` on failure.

#### List Files (Operation Code `5`)

1. **Client**:
   - Sends operation code `5`.
2. **Server**:
   - Sends the number of files (int32).
   - For each file, sends:
     - Filename length (int32) and filename.
     - File size (int64).
     - Last modified timestamp (int64).
   - Sends completion byte `0xFF`.

## API References

### Client Functions (`client.go`)

- 

main()

: Handles user interface and operation selection.
- 

authenticate(conn net.Conn) bool

: Manages user authentication.
- `uploadFile(filePath string) error`: Uploads a file to the server.
- `downloadFile(fileName string) error`: Downloads a file from the server.
- `viewFile(fileName string)`: Views the content of a file from the server.
- `deleteFile(fileName string)`: Deletes a file on the server.
- `listFiles()`: Lists all files in the user's directory on the server.

### Server Functions (`server.go`)

- 

main()

: Starts the server and listens for incoming connections.
- 

readCredentials(filePath string) (map[string]string, error)

: Reads user credentials from a file.
- 

handleConnection(conn net.Conn, credentials map[string]string, wg *sync.WaitGroup)

: Manages individual client connections.
- 

authenticate(conn net.Conn, credentials map[string]string) string

: Authenticates a client.
- 

handleClientOperations(conn net.Conn, username, clientDir string)

: Processes client operation requests.
- 

handleFileUpload(conn net.Conn, filePath string, fileSize int64, username string) error

: Handles file uploads.
- 

handleFileDownload(conn net.Conn, username string) error

: Handles file downloads.
- 

handleViewFile(conn net.Conn, filePath string, username string) error

: Handles file viewing.
- 

handleFileDeletion(reader *bufio.Reader, conn net.Conn, username string) error

: Handles file deletions.
- 

handleListFiles(conn net.Conn, clientDir string) error

: Handles listing files.
- 

handleShutdown(signalChannel chan os.Signal, wg *sync.WaitGroup)

: Gracefully shuts down the server on interrupt.

## Instructions for Future Enhancements

### Encryption

To enhance security, especially when transmitting sensitive data like authentication credentials and file content, encryption should be implemented.

#### Suggested Approach

- **TLS Encryption**:
  - Utilize Go's `crypto/tls` package to wrap the TCP connection with TLS.
  - Generate server certificates using a trusted CA or self-signed certificates for testing.
  - Update both client and server to establish a `tls.Conn` instead of a regular 

net.Conn

.

#### Implementation Steps

1. **Generate Certificates**:
   - Create a private key and certificate for the server.

2. **Update Server**:
   - Load the certificate and create a `tls.Config`.
   - Listen for TLS connections using `tls.Listen`.

   ```go
   cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
   config := &tls.Config{Certificates: []tls.Certificate{cert}}
   listener, err = tls.Listen("tcp", ":9090", config)
   ```

3. **Update Client**:
   - Configure TLS settings, possibly skipping certificate verification for self-signed certs (not recommended for production).

   ```go
   config := &tls.Config{InsecureSkipVerify: true}
   conn, err := tls.Dial("tcp", serverAddress, config)
   ```

4. **Test Encrypted Connection**:
   - Ensure that data transmitted between client and server is encrypted by using tools like Wireshark.

5. **Error Handling and Verification**:
   - Implement proper error checks and certificate verification to prevent man-in-the-middle attacks.

### Additional Enhancements

- **Improved Authentication**:
  - Integrate with a secure authentication system or database.
  - Implement account lockout policies after multiple failed attempts.

- **Concurrency Handling**:
  - Enhance the server to handle multiple client connections concurrently using Goroutines more effectively.

- **Logging and Monitoring**:
  - Implement comprehensive logging for auditing purposes.
  - Set up monitoring to track server health and performance.

- **File Transfer Optimization**:
  - Implement resume functionality for interrupted transfers.
  - Use compression to reduce bandwidth usage.

- **Client Improvements**:
  - Develop a GUI client for better user experience.
  - Add support for batch operations on multiple files.

---

By following this documentation and the instructions provided, future developers can understand the current system and implement the suggested enhancements effectively.