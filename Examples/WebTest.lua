package.path = package.path .. ";../?.so"
local web = require("goLuaWebserver")
local serverId = web.startWebserver("localhost:8080")

io.write("HTML:")
local html = io.read()
local function handleRequest(method, path)
	print(method)
	print(path)
    return  200, html, {["Content-Type"] = "text/html"}
end


local function handleSocket(client, messagetype, message)
	print(string.format("Client: %s MT: %d Message: %s",client, messagetype, message))
	web.writeWebSocket(serverId, client, message)
end

io.write("Path:")
local path = io.read()
web.serve(serverId, string.format("/%s",path), handleRequest)
web.serveWebSocket(serverId, "/ws", handleSocket)
io.read()
web.stopWebserver(serverId)
print("Stopping Server. Wait...")
io.read()
