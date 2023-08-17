package game

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ClientList map[*Client]bool

type Manager struct {
	clients ClientList
	games   map[string]*Game
	sync.RWMutex
}

type Message struct {
	Type      string `json:"type"`
	Client    string `json:"client"`
	Value     string `json:"value"`
	Key       int    `json:"key"`
	GameState Game   `json:"gameState"`
}

func StartManager() {
	manager := &Manager{clients: make(ClientList), games: make(map[string]*Game)}
	http.HandleFunc("/ws", manager.serveWebsocket)
}

func (m *Manager) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Connected")

	client := NewClient(conn, m)
	m.addClient(client)

	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}
}
