package.path = package.path .. ";../?.so"
local etlua = require("Examples.etlua")
local web = require("goLuaWebserver")
local serverId = web.startSecureWebserver("localhost:8080", "server.crt", "server.key")
local template = etlua.compile([[
  <body style="color: rgb(150,150,50); background: rgb(50,50,50);">
  <div>Hello <%= name %>,</div>
  Here are your items:
  <% for i, item in pairs(items) do %>
   <ul>* <%= i %> = <%= item %></ul>
  <% end %>
  </body>
]])
web.serveFiles(serverId, "/", [[./Examples/]])
io.write("Name:")
local name = io.read()
local function handleRequest(request)
	local everythang = {}
	for k, v in pairs(request) do
		if type(v) ~= "table" then
			everythang[k] = v
			print(string.format("%s = %s", k, v))
		end
		if type(v) == "table" then
			print(k)
			for k1, v1 in pairs(v) do
				if type(v1) == "string" then
					everythang[k1] = v1
					print(string.format("		%s = %s", k1, v1))
				end
			end
		end
	end
	local html = template({ name = name, items = everythang })
	return 200, html, { ["Content-Type"] = "text/html" }
end


local function handleSocket(client, messagetype, message)
	print(string.format("Client: %s MT: %d Message: %s", client, messagetype, message))
	web.writeWebSocket(serverId, client, message)
end

io.write("Path:")
local path = io.read()
web.serve(serverId, string.format("/%s", path), handleRequest)
web.serveWebSocket(serverId, "/ws", handleSocket)
io.read()
web.stopWebserver(serverId)
print("Stopping Server. Wait...")
io.read()
