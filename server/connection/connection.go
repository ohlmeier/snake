package connection

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ohlmeier/snake/game"
	"github.com/ohlmeier/snake/util"
)

type ClientList map[*Client]bool

var clientRooms map[string]string
var rooms map[string][]string
var games map[string]*game.Game

var manager Manager

type Client struct {
	connection *websocket.Conn

	manager *Manager

	egress chan []byte

	ID   string
	Room string
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte),
	}
}

type Manager struct {
	clients ClientList
	sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{clients: make(ClientList)}
}

type Message struct {
	Type      string    `json:"type"`
	Client    string    `json:"client"`
	Value     string    `json:"value"`
	Key       int       `json:"key"`
	GameState game.Game `json:"gameState"`
}

func Setup() {
	manager = NewManager()
	clientRooms = make(map[string]string)
	games = make(map[string]*game.Game)
	rooms = make(map[string][]string)
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", manager.serveWebsocket)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
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

func (c *Client) readMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()
	for {
		messageType, payload, err := c.connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
		log.Println("MessageType: ", messageType)
		log.Println("Payload: ", string(payload))
		var msg Message
		err = json.Unmarshal(payload, &msg)
		if err != nil {
			log.Println(err)
		}
		c.handleMsg(msg)
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:

			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println(err)
			}
			log.Println("sent message")
		}
	}
}

func (c *Client) handleMsg(msg Message) {
	switch msg.Type {
	case "newGame":
		c.handleNewGame(msg.Client)
	case "joinGame":
		c.handleJoinGame(msg.Client, msg.Value)
	case "keydown":
		c.handleKeydown(msg.Client, msg.Key)
	}
}

func (c *Client) handleNewGame(ID string) {
	log.Println("newGame called")
	roomName := util.RandStringRunes(5)
	clientRooms[ID] = roomName
	log.Println(ID)
	games[roomName] = game.New()
	games[roomName].AddPlayerOne(ID)
	rooms[roomName] = []string{ID}

	msg := Message{
		Type:  "gameCode",
		Value: roomName,
	}
	err := c.connection.WriteJSON(msg)
	if err != nil {
		log.Println(err)
	}
}

func (c *Client) handleJoinGame(ID, roomName string) {
	if val, exists := rooms[roomName]; exists {
		if len(val) == 0 {
			msg := Message{
				Type: "unknownCode",
			}
			err := c.connection.WriteJSON(msg)
			if err != nil {
				log.Println(err)
			}
		} else if len(val) > 1 {
			msg := Message{
				Type: "tooManyPlayers",
			}
			err := c.connection.WriteJSON(msg)
			if err != nil {
				log.Println(err)
			}
		}
	} else {
		msg := Message{
			Type: "unknownCode",
		}
		err := c.connection.WriteJSON(msg)
		if err != nil {
			log.Println(err)
		}
	}
	clientRooms[ID] = roomName
	log.Println(ID)

	games[roomName].AddPlayerTwo(ID)
	msg := Message{
		Type:  "gameCode",
		Value: roomName,
	}
	err := c.connection.WriteJSON(msg)
	if err != nil {
		log.Println(err)
	}
	for client, _ := range manager.clients {
		if client.Room == roomName {
			c.startGameInterval(roomName)
		}
	}
}

func (c *Client) handleKeydown(ID string, key int) {
	roomName := clientRooms[ID]
	log.Println(roomName)
	if roomName == "" {
		return
	}
	vel, err := game.GetUpdateVelocity(key)
	log.Printf("vel:\n%v", vel)
	if err != nil {
		log.Println(err)
	}
	if val, ok := games[roomName].Players[ID]; ok {
		val.Velocity = vel
		games[roomName].Players[ID] = val
	}
}

func (c *Client) startGameInterval(roomName string) {
	go func() {
		for {
			if _, exists := games[roomName]; !exists {
				return
			}
			winner := games[roomName].Loop()

			if winner == "" {
				emitGameState(conn, roomName, games[roomName])
			} else {
				emitGameOver(conn, roomName, winner)
				games[roomName] = nil
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

}

func emitGameState(conn *websocket.Conn, room string, game *game.Game) {
	msg := Message{
		Type:      "gameState",
		GameState: *game,
	}
	err := conn.WriteJSON(msg)
	if err != nil {
		fmt.Println(err)
	}
}

func emitGameOver(conn *websocket.Conn, room string, winner string) {
	msg := Message{
		Type:  "gameOver",
		Value: winner,
	}
	err := conn.WriteJSON(msg)
	if err != nil {
		fmt.Println(err)
	}
}
