local web = require("goLuaWebserver")
local GoLuaWeb = { servers = {}, nameCounter = 0, type = "GoLuaWeb" }
GoLuaWeb.__index = GoLuaWeb

local HTTPmethods = { "Post", "Get", "Put", "Patch", "Delete" }

for _, v in pairs(HTTPmethods) do
	GoLuaWeb[v] = function(self, options)
		GoLuaWeb.HttpMethods(self, options, string.upper(v))
	end
end

function GoLuaWeb:getNewName()
	GoLuaWeb.nameCounter = GoLuaWeb.nameCounter + 1
	return string.format("anonymouse_webserver_%d", GoLuaWeb.nameCounter)
end

function GoLuaWeb:HttpMethods(options, method)
	local id = self.id
	local path = options.path
	local func = options.func

	self.paths[path] = self.paths[path] or {}
	self.paths[path].methods = self.paths[path].methods or {}
	self.paths[path].methods[method] = self.paths[path].methods[method] or {}
	self.paths[path].methods[method].func = func

	if self.paths[path].success then
		return
	end

	local handlingFunc = function(request)
		local pathMethod = self.paths[path].methods[request.method]
		if pathMethod and pathMethod.func then
			return pathMethod.func(request)
		else
			return 404
		end
	end

	local success, errmsg = web.serve(id, path, handlingFunc)
	if not success then
		print(errmsg)
		return
	end
	self.paths[path].success = true
end

function GoLuaWeb:WebSocket(options)
	local id = self.id
	local path = options.path
	local func = options.func
	local method = "WEBSOCKET"

	self.paths[path] = self.paths[path] or {}
	self.paths[path].methods = self.paths[path].methods or {}
	self.paths[path].methods[method] = self.paths[path].methods[method] or {}
	self.paths[path].methods[method].func = func

	if self.paths[path].success then
		return
	end

	local handlingFunc = function(client, messagetype, message)
		local pathMethod = self.paths[path].methods[method]
		if pathMethod and pathMethod.func then
			pathMethod.func(client, messagetype, message)
		end
	end

	local success, errmsg = web.serveWebSocket(id, path, handlingFunc)
	if not success then
		print(errmsg)
		return
	end
	self.paths[path].success = true
end

function GoLuaWeb:new(cons)
	cons = cons or {}
	if cons.name and GoLuaWeb.servers[cons.name] then
		return GoLuaWeb.servers[cons.name]
	end
	local name = cons.name or GoLuaWeb:getNewName()
	local server = {}
	setmetatable(server, GoLuaWeb)

	local serverId = -1
	local errMsg = string.format("No server with name %s created.", name)
	if cons.secure then
		serverId, errMsg = web.startSecureWebserver(cons.address, cons.crt, cons.key)
	else
		serverId, errMsg = web.startWebserver(cons.address)
	end
	if serverId == -1 then
		print(string.format("Error when trying to create server '%s': %s", name, errMsg))
	end
	GoLuaWeb.servers[name] = server
	server.id = serverId
	self.paths = {}

	return server
end

return GoLuaWeb
