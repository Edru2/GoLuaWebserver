package.path = package.path .. ";../?.so"
GoLuaWeb = require "Examples.GoLuaWeb"


local function serveHallo()
	return 200, "Hello World!", { ["content-type"] = "text/html" }
end

local myserver = GoLuaWeb:new({ name = "myserver", address = ":8080" })

myserver:Get({ path = "/hallo", func = serveHallo })
io.read()
print("Changing function")
myserver:Get({ path = "/hallo", func = function() return 404 end })
io.read()
myserver:Post({ path = "/hallo", func = serveHallo })
io.read()
