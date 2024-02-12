#include "goLuaWebserver.h"
#include "libluaWebserver.h"
#include <lauxlib.h>
#include <lstate.h>
#include <lua.h>
#include <lualib.h>
#include <stdbool.h>
#include <string.h>

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
    getVerifiedArgument(L, 1, __func__, "adress", LUA_TSTRING, 0); /* Verify first argument is a string */
    const char* address = lua_tostring(L, 1);
    int serverId = StartServer(address);
    lua_pushinteger(L, serverId);
    return 1;
}

static int serve(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "path", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "hook function", LUA_TFUNCTION, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* path = lua_tostring(L, 2);

    lua_pushvalue(L, 3);
    int luaRef = luaL_ref(L, LUA_REGISTRYINDEX);
    Serve(L, serverId, path, luaRef);
    return 0;
}

static int serveWebSocket(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "path", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "hook function", LUA_TFUNCTION, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* path = lua_tostring(L, 2);
    lua_pushvalue(L, 3);
    int luaRef = luaL_ref(L, LUA_REGISTRYINDEX);
    ServeWebSocket(L, serverId, path, luaRef);
    return 0;
}

static int writeWebSocket(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    getVerifiedArgument(L, 2, __func__, "client", LUA_TSTRING, 0);
    getVerifiedArgument(L, 3, __func__, "message", LUA_TSTRING, 0);
    int serverId = luaL_checkinteger(L, 1);
    const char* client = lua_tostring(L, 2);
    const char* message = lua_tostring(L, 3);
    WriteToWebSocketClient(serverId, client, message);
    return 0;
}

static int stopWebserver(lua_State* L)
{
    getVerifiedArgument(L, 1, __func__, "server id", LUA_TNUMBER, 0);
    int serverId = luaL_checkinteger(L, 1);
    StopServer(serverId);
    lua_pushboolean(L, true);
    return 1;
}

LuaHttpResponse* callLuaFunc(lua_State* L, int luaRef, const char* method, const char* path)
{
    if (L == NULL || method == NULL || path == NULL) {
        return NULL;
    }
    LuaHttpResponse* response = (LuaHttpResponse*)malloc(sizeof(LuaHttpResponse));
    if (!response) {
        return NULL;
    }

    lua_settop(L, 0);
    lua_rawgeti(L, LUA_REGISTRYINDEX, luaRef);
    lua_pushstring(L, method);
    lua_pushstring(L, path);
    int error = 0;
    error = lua_pcall(L, 2, LUA_MULTRET, 0);
    if (error != 0) {
        const char* errorMsg = lua_tostring(L, -1);
        lua_pop(L, 1);  // Remove the error message from the stack
        free(response); // Avoid memory leak
        lua_pushfstring(L, "Error when calling hook function: %s", errorMsg);
        return NULL; // Indicate failure
    }

    if (lua_gettop(L) < 3) {
        free(response);
        lua_pushfstring(L, "Not enough arguments in function %s. Expected are %d got %d.", "http hook return", 3, lua_gettop(L));
        return NULL;
    }

    if (!lua_isnumber(L, 1)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 1, "status code", "integer", "http hook return");
        return NULL; // Indicate failure
    }
    if (!lua_isstring(L, 2)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 2, "response body", "string", "http hook return");
        return NULL; // Indicate failure
    }
    if (!lua_istable(L, 3)) {
        free(response);
        lua_pushfstring(L, "Argument %d (%s) must be a %s in function %s", 3, "header table", "table", "http hook return");
        return NULL; // Indicate failure
    }

    int statusCode = (int)lua_tonumber(L, 1);
    const char* responseBody = lua_tostring(L, -2);
    response->responseBody = malloc(strlen(responseBody) + 1);

    if (response->responseBody != NULL) {
        strcpy(response->responseBody, responseBody);
    }

    if (lua_istable(L, -1)) {
        lua_pushnil(L);
        int i = 0;
        while (lua_next(L, -2) != 0) {
            const char* headerName = lua_tostring(L, -2);
            const char* headerValue = lua_tostring(L, -1);

            if (i < 10) {
                strncpy(response->headersKeys[i], headerName, 255);
                response->headersKeys[i][255] = '\0';
                strncpy(response->headersValues[i], headerValue, 255);
                response->headersValues[i][255] = '\0';
            }

            lua_pop(L, 1);
            i++;
        }
        response->headersCount = i;
    }
    response->statusCode = statusCode;

    lua_pop(L, 3);
    return response;
}

void callLuaWebSocketFunc(lua_State* L, int luaRef, char* client, int messagetype, char* message)
{
    if (L == NULL || message == NULL) {
        return;
    }

    lua_settop(L, 0);
    lua_rawgeti(L, LUA_REGISTRYINDEX, luaRef);
    lua_pushstring(L, client);
    lua_pushinteger(L, messagetype);
    lua_pushstring(L, message);
    free(message);
    free(client);
    int error = 0;
    error = lua_pcall(L, 3, 0, 0);
    if (error != 0) {
        const char* errorMsg = lua_tostring(L, -1);
        lua_pop(L, 1);
        printf("%s\n", errorMsg);
        return;
    }
}

int luaopen_goLuaWebserver(lua_State* L)
{
    createMyResource(L);
    lua_newtable(L);
    lua_pushcfunction(L, startWebserver);
    lua_setfield(L, -2, "startWebserver");
    lua_pushcfunction(L, stopWebserver);
    lua_setfield(L, -2, "stopWebserver");
    lua_pushcfunction(L, serve);
    lua_setfield(L, -2, "serve");
    lua_pushcfunction(L, serveWebSocket);
    lua_setfield(L, -2, "serveWebSocket");
    lua_pushcfunction(L, writeWebSocket);
    lua_setfield(L, -2, "writeWebSocket");

    return 1;
}
