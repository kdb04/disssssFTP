Distributed File Orchestration and Synchronization: Multi-Node Data-Transfer-Framework for Linux
Overview:

Design and implement a multi-client file transfer system using a client-server model in C, python, or Golang, where the server can handle multiple clients simultaneously. The server should authenticate clients, allow them to upload, download, view, and delete files from a server-side directory, and respond to multiple concurrent requests without crashing or losing data.
Background:

Modern file transfer systems, such as FTP, are widely used for transferring data between clients and servers in networked environments. Such systems need to support various functionalities like user authentication, file uploading, downloading, and secure data management. Additionally, the system must handle multiple clients simultaneously without sacrificing performance or data integrity.
Problem Statement:

You are tasked with implementing a multi-client file transfer system using sockets(this is mandatory) that will:

    Authenticate clients based on a predefined list of usernames and passwords.
    Allow authenticated clients to perform the following actions:
        Upload Files: Clients should be able to upload files to their specific directory on the server.
        Download Files: Clients should be able to download files that exist in their directory on the server.
        View Files: Clients can preview the first 1024 bytes of any file in their directory.
        Delete Files: Clients can delete any file within their directory on the server.
        List Files: Clients should be able to request a list of all files stored in their specific directory.
    Handle multiple clients concurrently, without interference between them.
    Support a robust signal handling mechanism that ensures the server can safely shut down while maintaining data integrity.

Functional Requirements:

    Client Authentication:
        The server maintains a list of valid username-password pairs in a file (id_passwd.txt).
        Upon connecting, clients must provide a valid username and password. If the credentials are incorrect, the server should reject the client and terminate the session.
    File Management:
        Clients should be able to:
            Upload Files: After authenticating, a client can upload a file to the server by providing the file name. The server should save the file in a directory specific to the client (e.g., /server_storage/<username>).
            Download Files: The client can request a file to download from their directory on the server. If the file exists, the server should send it; otherwise, an error message is returned.
            View Files: The client can request a preview of the first 1024 bytes of any file in their directory.
            Delete Files: The client can delete any file from their directory. Upon successful deletion, the server should confirm the operation.
            List Files: Clients can request a list of all files stored in their directory. The server should send a list of file names in that directory.
    Concurrency and Resource Management:
        The server must handle multiple clients simultaneously. When a new client connects, the server should fork a new process or use threading (if desired) to handle the client’s requests independently, without blocking other clients from connecting.
        The server must handle signals (e.g., SIGINT) to close open sockets gracefully and ensure that any active client connections are properly terminated upon server shutdown.
    Robust Error Handling:
        Ensure that errors such as file not found, invalid commands, or communication failures are handled gracefully and communicated clearly to the client.
        Invalid file paths should not allow clients to escape their designated directories or access unauthorized files.
    Security and Data Integrity:
        The system should ensure that clients can only access their own files. Each client should have a dedicated directory (e.g., /server_storage/<username>) that no other client can access.
        Handle possible race conditions or conflicts that could arise from multiple clients accessing the server concurrently.
        Consider potential security risks such as buffer overflows or directory traversal attacks and mitigate them accordingly.

image
Non-Functional Requirements:

    Scalability:
        The system should be designed to handle at least 2 concurrent clients without noticeable degradation in performance.
    Maintainability:
        The code should be modular, well-commented, and designed in a way that additional features like encryption or more complex authentication mechanisms can be added easily in the future.
    Reliability:
        The server must guarantee that file uploads and downloads are completed fully and accurately, even if multiple clients are performing these actions at the same time.
        If the server crashes or is interrupted (e.g., by a SIGINT signal), all resources (sockets, file descriptors) should be closed properly.
    Efficiency:
        The server should minimize CPU and memory usage, even when handling multiple clients.
        Data transfers between clients and the server should be optimized to minimize delays, especially for large files.

Constraints:

    Programming Language: The server must be implemented in C, Python, or Golang. (suggested to use Unix-based socket programming).
    Platform: The system will be deployed on a Unix-based system (e.g., Linux).
    Security: For this phase of development, simple authentication (username/password) without encryption is acceptable, but the system must be designed with future security improvements in mind.

Assumptions:

    The server will have access to a file (id_passwd.txt) that contains valid username-password pairs for client authentication.
    All clients accessing the server will use the same protocol for communicating with the server (TCP-based).
    Each client will have their own directory on the server where files are stored (e.g., /server_storage/<username>).

Challenges:

    Concurrency: Managing multiple clients concurrently, ensuring that one client’s file operations do not interfere with another’s.
    Data Integrity: Guaranteeing that file transfers are complete and accurate, even in the case of concurrent uploads/downloads.
    Error Handling: Handling unexpected conditions (e.g., invalid commands, missing files) gracefully and without crashing the server.
    Security: Preventing unauthorized access to files and directories through proper path management and validation.
    Signal Handling: Ensuring that the server can handle signals like SIGINT (Ctrl+C) for safe shutdown, closing all open sockets, and ensuring data consistency.

💡 Signal Handling: To ensure proper shutdown of a server when it receives termination signals such as `SIGINT` (triggered when the client presses `Ctrl+C`), you need to implement a signal handler in your server code. This handler will gracefully close the server's resources (like sockets) before shutting down the server.
Expected Outcome:

By the end of the project, the server should be able to:

    Authenticate multiple clients concurrently.
    Allow each authenticated client to upload, download, view, and delete files in their directory.
    List the contents of a client’s directory.
    Safely handle multiple concurrent connections without data loss or corruption.
    Shut down gracefully upon receiving an interrupt signal, ensuring all resources are properly cleaned up.

Deliverables
Week 1: Network Infrastructure & Authentication Layer

    Client-Server Network Infrastructure:
        Design and implement TCP-based socket communication with a robust handshake protocol for initializing connections.
        Socket binding and efficient management of I/O buffers to handle multiple concurrent data streams.

    Authentication Subsystem:
        Develop an authentication mechanism integrated with the socket layer, reading from a secure credential store (id_passwd.txt).
        Handle session persistence to ensure continuous authentication for long-duration file transfer sessions.

    File Upload Protocol:
        Implement an initial version of the file upload protocol, with client-side segmentation for large files and server-side error correction during transmission.
        Store uploaded files in a hierarchically organized directory structure, ensuring user isolation.

    Signal Handling for Controlled Shutdown:
        Build an initial signal management system, intercepting termination signals (like SIGINT), ensuring clean socket closure, and maintaining connection integrity.

Complex Goals:

    Establish a robust client-server infrastructure, secure file uploads, and a controlled shutdown process capable of handling multiple concurrent sessions.

Week 2: Advanced Concurrency and File Management System

    Concurrency Management:
        Implement a multi-threaded or multi-process server architecture to handle parallel client requests, focusing on non-blocking I/O operations and dynamic memory management.
        Employ a thread-pooling or process-forking strategy to reduce resource contention and optimize CPU scheduling under high load.

    Enhanced File Download & View Protocols:
        Develop a file download mechanism to allow clients to download files.
        Implement the byte-range file viewing system, allowing clients to preview partial file contents without loading full files into memory (preview the first 1024 bytes of any file in their directory).

    File Deletion & Directory Listing with Concurrency Control:
        Implement a way for file deletion and dynamically generating/listing directory contents while ensuring isolation across concurrent client requests.

    Comprehensive Error and Exception Handling:
        Integrate a detailed error-handling framework for the system, addressing potential edge cases such as invalid commands, partial file transfers, and directory traversal attempts.
        Log errors if possible with detailed diagnostics for post-operation auditing and debugging.

Complex Goals:

    Achieve efficient concurrency, supporting multiple simultaneous file operations, and build comprehensive error handling for real-time system reliability.

Week 3: Security, Testing, and System Hardening

    Security Reinforcements:
        Implement directory isolation with enhanced file path validation to prevent directory traversal attacks and ensure clients remain confined to their respective directories.
        Conduct security audits on file operations, identifying and mitigating risks like buffer overflow vulnerabilities and unauthorized file access.

    System-wide Stress Testing:
        Simulate high-load scenarios with multiple concurrent clients to evaluate system performance and stability, focusing on latency, data throughput, and CPU/memory utilization under stress.
        Introduce signal-driven interruptions during active file transfers, ensuring that the server can handle shutdown signals without data corruption.

    Code Modularity and Documentation:
        Modularize code into well-defined components for authentication, file management, network handling, and error processing, promoting maintainability.
        Document critical system components, including API references, protocol specifications, and instructions for future enhancements like encryption.

    Optional Enhancements:
        Explore additional security features such as encrypted client-server communication using SSL/TLS.
        Develop logging and auditing subsystems for client activity tracking, focusing on upload/download transactions and system-level alerts.
