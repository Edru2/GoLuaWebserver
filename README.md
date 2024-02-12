# Lua-Go Webserver

The Lua-Go Webserver Integration project enables the use of Lua scripts to manage and control web server built in Go. This unique combination allows developers to leverage the simplicity and flexibility of Lua scripting for server-side logic, alongside Go's robust networking capabilities, efficient concurrency model, and high scalability. Whether serving dynamic HTTP content or handling real-time communications through WebSockets, this integration offers a powerful and versatile platform for web application development.

## Features

-   **HTTP Server**: Serve HTTP requests with custom Lua scripts, enabling dynamic response generation based on the request details.
-   **WebSocket Support**: Establish real-time bi-directional communication channels between the server and clients using WebSockets, with Lua scripts handling the data exchange.
-   **Lua Scripting**: Leverage the flexibility of Lua for writing server logic, including request processing, response generation, and handling WebSocket messages.
-   **Multi-Path Support**: Configure multiple paths with distinct Lua handlers for diversified request handling within a single server instance.
-   **Concurrent Connections**: Benefit from Go's goroutine-based concurrency model to handle multiple connections efficiently, making the server scalable and responsive.

## Getting Started

### Prerequisites

-   Go (version 1.13 or higher recommended)
-   Lua 5.1 development libraries
-   GCC compiler for CGo integration

### Installation

1.  **Clone the Repository**
```bash
git clone https://github.com/edru2/GoLuaWebserver
cd GoLuaWebserver
```
-   **Build the Project**
    
    Use the Go toolchain to build the server binary. This process will compile both the Go and Lua parts of the project.
    
```bash
make clean
make
```   

### Running the Server

**Start the Server**
    
After compiling, start the server by including the Lua-Go webserver library in your Lua script. Below is a sample Lua script demonstrating how to use the API to create a web server with dynamic HTTP and WebSocket endpoints.
 
 #### Example Lua Script

```lua

local web = require("goLuaWebserver")

-- Start the server on localhost port 8080
local serverId = web.startWebserver("localhost:8080")

-- HTML content to serve for example <h1> hello world </h1>
io.write("HTML content to serve: ")
local html = io.read()

-- HTTP request handler
local function handleRequest(method, path)
    print("HTTP Request Method:", method)
    print("HTTP Request Path:", path)
    return 200, html, {["Content-Type"] = "text/html"}
end

-- WebSocket message handler
local function handleSocket(client, messageType, message)
    print(string.format("WebSocket Client: %s, Message Type: %d, Message: %s", client, messageType, message))
    -- Echoes received message back to the client
    web.writeWebSocket(serverId, client, message)
end

-- Register the HTTP and WebSocket handlers
io.write("Path for HTTP handler: ")
local path = io.read()
web.serve(serverId, "/" .. path, handleRequest)
web.serveWebSocket(serverId, "/ws", handleSocket)

-- Cleanup and stop the server
io.read() -- Wait for user input to proceed
web.stopWebserver(serverId)
print("Server stopped. Press any key to exit...")
io.read()
 ``` 
    
### API Overview

-   **`web.startWebserver(address)`**: Initializes and starts an HTTP/WebSocket server listening on the specified address, such as `"localhost:8080"`. This function returns a unique `serverId` which is used to identify the server in subsequent API calls.
    
-   **`web.serve(serverId, path, handlerFunction)`**: Registers a Lua function as a handler for HTTP requests targeting a specific path. The `handlerFunction` must follow a specific signature, accepting `method` and `path` as parameters, and it must return three values:
    
    -   An HTTP status code (e.g., `200` for OK).
    -   A response body, which is the content to be sent back to the client.
    -   A table of headers, where each key-value pair represents a single header field and its value.
    
    This setup allows for dynamic response generation based on the request details, enabling developers to implement a wide range of web functionalities.
    
-   **`web.serveWebSocket(serverId, path, handlerFunction)`**: Registers a Lua function as a handler for WebSocket connections established on a specified path. The `handlerFunction` is invoked with parameters `client`, `messageType`, and `message` whenever a WebSocket message is received. This allows the server to handle real-time data exchange through WebSockets, facilitating interactive web applications.
    
-   **`web.writeWebSocket(serverId, client, message)`**: Sends a message to a specific WebSocket client. This function requires the `serverId` to identify the server, `client` to specify the recipient client, and `message` as the content to be sent. It is often used within WebSocket handler functions to implement bidirectional communication.
    
-   **`web.stopWebserver(serverId)`**: Stops the specified web server identified by `serverId` and releases all associated resources. This function is essential for clean shutdown procedures, ensuring that all server activities are gracefully terminated before the application exits.

## Contributing

Contributions are welcome! Please feel free to submit pull requests, report bugs, and suggest features through the project's issue tracker.

## License

This project is licensed under the MIT License
