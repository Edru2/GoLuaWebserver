#include "goLuaWebserver.h"
#include <lauxlib.h>
#include <lua.h>
#include <lualib.h>
#include <stdbool.h>

typedef struct
{
} LuaAliveStruct;

static int myResourceGC(lua_State* L)
{
    LuaAliveStruct* resource = (LuaAliveStruct*)lua_touserdata(L, 1);
    StopLuaStateFunctions(L);
    return 0;
}

static void setUserdataMetatable(lua_State* L)
{
    luaL_newmetatable(L, "MyResourceMeta");
    lua_pushstring(L, "__gc");
    lua_pushcfunction(L, myResourceGC);
    lua_settable(L, -3);
}

static const char* USERDATA_KEY = "IsLuaStateStillAlive?";
static int createMyResource(lua_State* L)
{
    LuaAliveStruct* resource = (LuaAliveStruct*)lua_newuserdata(L, sizeof(LuaAliveStruct));
    setUserdataMetatable(L); // Call this to set the metatable for the userdata
    lua_setmetatable(L, -2); // Set the metatable for the userdata on the stack
    lua_pushlightuserdata(L, (void*)&USERDATA_KEY);
    lua_pushvalue(L, -2);               // Copy userdata to the top
    lua_settable(L, LUA_REGISTRYINDEX); // Stores the copy in the registry
    lua_settop(L, 0);
    return 0;
}
/* Error handling function */
void errorArgumentType(lua_State* L, const char* functionName, int pos, const char* publicName, const char* typeName, int isOptional)
{
    if (!isOptional) {
        luaL_error(L, "Argument %d (%s) must be a %s in function %s", pos, publicName, typeName, functionName);
    }
}

/* Generalized argument verification function */
void getVerifiedArgument(lua_State* L, int pos, const char* functionName, const char* publicName, int type, int isOptional)
{
    switch (type) {
    case LUA_TSTRING:
        if (!lua_isstring(L, pos)) {
            errorArgumentType(L, functionName, pos, publicName, "string", isOptional);
        }
        break;
    case LUA_TNUMBER:
        if (!lua_isnumber(L, pos)) {
            errorArgumentType(L, functionName, pos, publicName, "integer", isOptional);
        }
        break;
    case LUA_TTABLE:
        if (!lua_istable(L, pos)) {
            errorArgumentType(L, functionName, pos, publicName, "table", isOptional);
        }
        break;

    case LUA_TFUNCTION:
        if (!lua_isfunction(L, pos)) {
            errorArgumentType(L, functionName, pos, publicName, "function", isOptional);
        }
        break;

    default:
        luaL_error(L, "Unsupported type %d for verification in function %s", type, functionName);
    }
}

static int startWebserver(lua_State* L)
{
    Message message;
    getVerifiedArgument(L, 1, __func__, "address", LUA_TSTRING, 0);
    const char* address = lua_tostring(L, 1);
    message = StartServer(address, NULL, NULL);
    if (!message.success) {
        lua_pushinteger(L, -1);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    lua_pushinteger(L, message.id);
    return 1;
}

static int startSecureWebserver(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "address", LUA_TSTRING, 0);
    getVerifiedArgument(L, 2, __func__, "certFile", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "keyFile", LUA_TSTRING, 0);
    const char* address = lua_tostring(L, 1);
    const char* certFile = lua_tostring(L, 2);
    const char* keyFile = lua_tostring(L, 3);
    Message message = StartServer(address, certFile, keyFile);
    if (!message.success) {
        lua_pushinteger(L, -1);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    lua_pushinteger(L, message.id);
    return 1;
}

static int serve(lua_State* L)
{
    Message message;
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "path", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "hook function", LUA_TFUNCTION, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* path = lua_tostring(L, 2);

    lua_pushvalue(L, 3);
    int luaRef = luaL_ref(L, LUA_REGISTRYINDEX);
    message = Serve(L, serverId, path, luaRef);
    if (!message.success) {
        lua_pushboolean(L, false);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    lua_pushboolean(L, true);
    return 1;
}

static int serveWebSocket(lua_State* L)
{
    Message message;
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "path", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "hook function", LUA_TFUNCTION, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* path = lua_tostring(L, 2);
    lua_pushvalue(L, 3);
    int luaRef = luaL_ref(L, LUA_REGISTRYINDEX);
    message = ServeWebSocket(L, serverId, path, luaRef);
    if (!message.success) {
        lua_pushboolean(L, false);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }

    lua_pushboolean(L, true);
    return 1;
}

static int writeWebSocket(lua_State* L)
{
    Message message;
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "client", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "message", LUA_TSTRING, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* client = lua_tostring(L, 2);
    const char* sendMessage = lua_tostring(L, 3);
    message = WriteToWebSocketClient(serverId, client, sendMessage);
    if (!message.success) {
        lua_pushboolean(L, false);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    lua_pushboolean(L, true);
    return 1;
}

static int serveFiles(lua_State* L)
{
    Message message;
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "path", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "directory", LUA_TSTRING, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* path = lua_tostring(L, 2);
    const char* dir = lua_tostring(L, 3);
    message = ServeFiles(serverId, path, dir);
    if (!message.success) {
        lua_pushboolean(L, false);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    lua_pushboolean(L, true);
    return 1;
}

static int stopWebserver(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    int serverId = luaL_checkinteger(L, 1);
    Message message = StopServer(serverId);
    if (!message.success) {
        lua_pushboolean(L, false);
        lua_pushstring(L, message.msg);
        free(message.msg);
        return 2;
    }
    return 0;
}

#ifndef LUAWEBSERVER_LIB
#ifdef _WIN32
#define LUAWEBSERVER_LIB __declspec(dllexport)
#else
#define LUAWEBSERVER_LIB __attribute__((visibility("default")))
#endif
#endif

LUAWEBSERVER_LIB int luaopen_goLuaWebserver(lua_State* L)
{
    createMyResource(L);
    lua_newtable(L);
    lua_pushcfunction(L, startWebserver);
    lua_setfield(L, -2, "startWebserver");
    lua_pushcfunction(L, startSecureWebserver);
    lua_setfield(L, -2, "startSecureWebserver");
    lua_pushcfunction(L, stopWebserver);
    lua_setfield(L, -2, "stopWebserver");
    lua_pushcfunction(L, serve);
    lua_setfield(L, -2, "serve");
    lua_pushcfunction(L, serveWebSocket);
    lua_setfield(L, -2, "serveWebSocket");
    lua_pushcfunction(L, writeWebSocket);
    lua_setfield(L, -2, "writeWebSocket");
    lua_pushcfunction(L, serveFiles);
    lua_setfield(L, -2, "serveFiles");

    return 1;
}
