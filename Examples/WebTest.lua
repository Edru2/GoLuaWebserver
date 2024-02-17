package.path = package.path .. ";../?.so"
local web = require("goLuaWebserver")
local serverId = web.startWebserver("localhost:8080")

io.write("HTML:")
local html = io.read()
local function handleRequest(request)
	for k,v in pairs(request) do
		if type(v) ~= "table" then
			print(string.format("%s = %s", k, v))
		end
		if type(v) == "table" then
				print(k)
				for k1,v1 in pairs(v) do
		if type(v1) == "string" then
					print(string.format("		%s = %s", k1, v1))
		end
	end

		end
	end
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
