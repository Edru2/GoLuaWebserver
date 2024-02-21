package.path = package.path .. ";../?.so"
local etlua = require("Examples.etlua")
local web = require("goLuaWebserver")
local open = io.open

local function readFile(path)
	local file = open(path, "rb") -- r read mode and b binary mode
	if not file then return nil end
	local content = file:read "*a" -- *a or *all reads the whole file
	file:close()
	return content
end

local function clickedFunc(response)
	local file = readFile("./Examples/htmxExample/table.etlua")
	local etluaCont = etlua.compile(file)
	return 200, etluaCont({name = "Edru", items = response}), { ["content-type"] = "text/html" }
end

local serverId = web.startSecureWebserver("localhost:8080", "server.crt", "server.key")

web.serveFiles(serverId, "/", [[./Examples/htmxExample/]])

web.serve(serverId, "/clicked", clickedFunc)

local function handleSocket(client, messagetype, message)
	print(string.format("Client: %s MT: %d Message: %s", client, messagetype, message))
	web.writeWebSocket(serverId, client, message)
end

web.serveWebSocket(serverId, "/ws", handleSocket)
io.read()
web.stopWebserver(serverId)
print("Stopping Server. Wait...")
io.read()
