package main

/*
#cgo CFLAGS: -I./external/luajit/src
#cgo LDFLAGS: -L. -lluaWebserverHelper
#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
#include "LuaWebserverHelper.h"
#include <stdlib.h>
#include <string.h>
typedef const char cchar_t;
*/
import "C"
import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
}

type PathFunction struct {
	FunctionRef C.int
	LuaState    *C.lua_State
	Clients     map[string]*Client
}

type Server struct {
	Paths            map[string]*PathFunction
	WebSocketClients map[string]*Client
	http.Server
}

var (
	servers = make(map[int]*Server)
	nextID  = 0
	mutex   = &sync.RWMutex{}
)

func generateUniqueID() string {
	return uuid.New().String()
}

//export Serve
func Serve(L *C.lua_State, serverID C.int, path *C.cchar_t, luaFuncRef C.int) {
	goServerId := int(serverID)

	mutex.RLock()
	server, exists := servers[goServerId]
	mutex.RUnlock()
	if !exists {
		log.Printf("Server with ID %d not found", serverID)
		return
	}

	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()
	if exists {
		log.Printf("Path exists already!")
		return
	}

	mutex.Lock()
	server.Paths[goPath] = &PathFunction{
		FunctionRef: luaFuncRef,
		LuaState:    L,
	}
	mutex.Unlock()

	mux := server.Handler.(*http.ServeMux)
	mux.HandleFunc(goPath,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != goPath {
				http.NotFound(w, r)
				return
			}
			luaState := server.Paths[goPath].LuaState
			luaFuncRef := server.Paths[goPath].FunctionRef
			if luaState == nil {
				return
			}
			mutex.Lock()
			statusCode, responseBody, headers := callLuaFunction(luaState, luaFuncRef, r, goPath)
			mutex.Unlock()
			for key, value := range headers {
				w.Header().Set(key, value)
			}
			w.WriteHeader(statusCode)
			w.Write([]byte(responseBody))
		})
}

func callLuaFunction(L *C.lua_State, luaFuncRef C.int, r *http.Request, path string) (int, string, map[string]string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	bodyContent := string(bodyBytes)
	cReq := C.HttpRequest{
		method:        C.CString(r.Method),
		path:          C.CString(path),
		url:           C.CString(r.URL.String()),
		proto:         C.CString(r.Proto),
		contentLength: C.long(r.ContentLength),
		host:          C.CString(r.Host),
		remoteAddr:    C.CString(r.RemoteAddr),
		headersCount:  C.int(len(r.Header)),
		body:          C.CString(bodyContent),
	}

	defer func() {
		C.free(unsafe.Pointer(cReq.method))
		C.free(unsafe.Pointer(cReq.path))
		C.free(unsafe.Pointer(cReq.url))
		C.free(unsafe.Pointer(cReq.proto))
		C.free(unsafe.Pointer(cReq.host))
		C.free(unsafe.Pointer(cReq.remoteAddr))
		C.free(unsafe.Pointer(cReq.body))
	}()
	maxHeaders := 20
	if len(r.Header) < maxHeaders {
		maxHeaders = len(r.Header)
	}
	cReq.headersCount = C.int(maxHeaders)

	i := 0
	for key, values := range r.Header {
		if i >= maxHeaders {
			break
		}
		keyCStr := C.CString(key)
		valueCStr := C.CString(strings.Join(values, "|"))
		defer C.free(unsafe.Pointer(keyCStr))
		defer C.free(unsafe.Pointer(valueCStr))

		C.strncpy(&cReq.headersKeys[i][0], keyCStr, 255)
		cReq.headersKeys[i][255] = 0 // Ensure null termination
		C.strncpy(&cReq.headersValues[i][0], valueCStr, 255)
		cReq.headersValues[i][255] = 0 // Ensure null termination
		i++
	}
	///////
	cResponse := C.callLuaFunc(L, luaFuncRef, &cReq)
	///////
	responseHeaders := make(map[string]string)

	if cResponse == nil {
		errMsg := C.GoString(C.lua_tolstring(L, -1, nil))
		return 500, fmt.Sprintf("Internal Server Error: %s", errMsg), responseHeaders
	}

	defer C.free(unsafe.Pointer(cResponse))
	statusCode := int(cResponse.statusCode)
	var responseBody string
	if cResponse.responseBody != nil {
		defer C.free(unsafe.Pointer(cResponse.responseBody))
		responseBody = C.GoString(cResponse.responseBody)
	}
	for i := 0; i < int(cResponse.headersCount); i++ {
		key := C.GoString(&cResponse.headersKeys[i][0])
		value := C.GoString(&cResponse.headersValues[i][0])
		responseHeaders[key] = value
	}

	return statusCode, responseBody, responseHeaders
}

//export StartServer
func StartServer(address *C.cchar_t) C.int {
	serverAddress := C.GoString(address)
	id := nextID
	nextID++
	mux := http.NewServeMux()
	server := &Server{
		Server: http.Server{
			Addr:           serverAddress,
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Paths:            make(map[string]*PathFunction),
		WebSocketClients: make(map[string]*Client),
	}
	servers[id] = server
	log.Println("Opening new webserver at:", serverAddress)
	go func() {
		defer delete(servers, id)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("ListenAndServe: ", err)
		}
	}()
	return C.int(id)
}

//export ServeWebSocket
func ServeWebSocket(L *C.lua_State, serverID C.int, path *C.cchar_t, luaFuncRef C.int) {
	goServerId := int(serverID)

	server, exists := servers[goServerId]
	if !exists {
		log.Printf("Server with ID %d not found", serverID)
		return
	}
	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()

	if exists {
		log.Printf("Path exists already!")
		return
	}
	mutex.Lock()
	server.Paths[goPath] = &PathFunction{
		FunctionRef: luaFuncRef,
		LuaState:    L,
		Clients:     make(map[string]*Client),
	}
	mutex.Unlock()

	mux := server.Handler.(*http.ServeMux)

	var upgrader = websocket.Upgrader{}
	mux.HandleFunc(goPath,
		func(w http.ResponseWriter, r *http.Request) {
			luaState := server.Paths[goPath].LuaState
			luaFuncRef := server.Paths[goPath].FunctionRef
			if luaState == nil {
				return
			}
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Print("upgrade:", err)
				return
			}
			defer c.Close()

			clientID := generateUniqueID()
			client := Client{ID: clientID, Conn: c}
			mutex.Lock()
			server.WebSocketClients[clientID] = &client
			server.Paths[goPath].Clients[clientID] = &client
			mutex.Unlock()
			defer delete(server.Paths[goPath].Clients, clientID)
			defer delete(server.WebSocketClients, clientID)

			for {
				mt, message, err := c.ReadMessage()
				if mt == websocket.CloseMessage {
					log.Printf("Client: %s disconnected.", clientID)
					break
				}
				if err != nil {
					log.Println("read:", err)
					break
				}
				mutex.Lock()
				C.callLuaWebSocketFunc(luaState, luaFuncRef, C.CString(clientID), C.int(mt), C.CString(string(message)))
				mutex.Unlock()
			}
		})
}

//export WriteToWebSocketClient
func WriteToWebSocketClient(serverID C.int, clientID *C.cchar_t, message *C.cchar_t) {
	goClientId := C.GoString(clientID)
	goServerId := int(serverID)
	goMessage := C.GoString(message)

	server, exists := servers[goServerId]
	if !exists {
		log.Printf("Server with ID %d not found", goServerId)
		return
	}

	client, exists := server.WebSocketClients[goClientId]
	if !exists {
		log.Printf("Client with ID %s not found", goClientId)
		return
	}

	err := client.Conn.WriteMessage(1, []byte(goMessage))
	if err != nil {
		log.Println("write:", err)
	}

}

//export ServeFiles
func ServeFiles(serverID C.int, path *C.cchar_t, dir *C.cchar_t) {
	goServerId := int(serverID)
	server, exists := servers[goServerId]
	if !exists {
		log.Printf("Server with ID %d not found", serverID)
		return
	}
	goDir := C.GoString(dir)
	goPath := C.GoString(path)
	if goPath == "" {
		goPath = "/"
	}
	mutex.RLock()
	_, exists = server.Paths[goPath]
	mutex.RUnlock()
	if exists {
		log.Printf("Path exists already!")
		return
	}

	mutex.Lock()
	server.Paths[goPath] = &PathFunction{}
	mutex.Unlock()

	fileServer := http.FileServer(http.Dir(goDir))
	mux := server.Handler.(*http.ServeMux)
	mux.Handle(goPath, http.StripPrefix(goPath, fileServer))
}

//export StopServer
func StopServer(serverID C.int) {
	id := int(serverID)
	GoStopServer(id)
}

func GoStopServer(id int) {
	server, exists := servers[id]
	if exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for clientID, client := range server.WebSocketClients {
			err := client.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("Failed to write close message to client %s: %v", clientID, err)
			}
			err = client.Conn.Close()
			if err != nil {
				log.Printf("Failed to close client %s: %v", clientID, err)
			}
		}
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Shutdown failed: %+v", err)
			GoStopServer(id)
		} else {
			delete(servers, id)
			log.Printf("Server with ID %d shut down successfully", id)
		}
	} else {
		log.Printf("Server with ID %d not found", id)
	}
}

//export StopAllServers
func StopAllServers() {
	for id := range servers {
		GoStopServer(id)
	}
}

//export StopLuaStateFunctions
func StopLuaStateFunctions(L *C.lua_State) {
	for _, server := range servers {
		for _, path := range server.Paths {
			if L == path.LuaState {
				path.FunctionRef = 0
				path.LuaState = nil
			}
		}
	}
}

func main() {}
