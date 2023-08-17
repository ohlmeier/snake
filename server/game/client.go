package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ohlmeier/snake/util"
)

type Client struct {
	connection *websocket.Conn

	manager *Manager

	egress chan []byte

	Number int
	Room   string
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte),
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
					log.Println("router closed: ", err)
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
		c.handleNewGame()
	case "joinGame":
		c.handleJoinGame(msg.Value)
	case "keydown":
		c.handleKeydown(msg.Key)
	}
}

func (c *Client) handleNewGame() {
	log.Println("newGame called")
	roomName := util.RandStringRunes(5)
	c.manager.games[roomName] = New()
	c.manager.games[roomName].AddPlayerOne()

	c.Number = 1
	c.Room = roomName

	msg := Message{
		Type:  "gameCode",
		Value: roomName,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	c.egress <- data

	/*err := c.connection.WriteJSON(msg)
	if err != nil {
		log.Println(err)
	}*/
}

func (c *Client) handleJoinGame(roomName string) {
	if g, exists := c.manager.games[roomName]; exists {
		if g.IsFull() {
			msg := Message{
				Type: "tooManyPlayers",
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
				return
			}
			c.egress <- data
		}
	} else {
		msg := Message{
			Type: "unknownCode",
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Println(err)
			return
		}
		c.egress <- data
	}
	c.Number = 2
	c.Room = roomName

	c.manager.games[roomName].AddPlayerTwo()
	msg := Message{
		Type:  "gameCode",
		Value: roomName,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	c.egress <- data
	for client, _ := range c.manager.clients {
		if client.Room == roomName {
			c.startGameInterval(roomName)
		}
	}
}

func (c *Client) handleKeydown(key int) {
	if c.Room == "" {
		return
	}
	vel, err := getUpdateVelocity(key)
	log.Printf("vel:\n%v", vel)
	if err != nil {
		log.Println(err)
	}
	if len(c.manager.games[c.Room].Players) > 1 {
		c.manager.games[c.Room].Players[c.Number-1].Velocity = vel
	}
}

func (c *Client) startGameInterval(roomName string) {
	go func() {
		for {
			if _, exists := c.manager.games[roomName]; !exists {
				return
			}
			winner := c.manager.games[roomName].Loop()

			if winner == "" {
				c.emitGameState(roomName, c.manager.games[roomName])
			} else {
				c.emitGameOver(roomName, winner)
				c.manager.games[roomName] = nil
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

}

func (c *Client) emitGameState(room string, game *Game) {
	c.manager.Lock()
	defer c.manager.Unlock()
	msg := Message{
		Type:      "gameState",
		GameState: *game,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	c.egress <- data
}

func (c *Client) emitGameOver(room string, winner string) {
	c.manager.Lock()
	defer c.manager.Unlock()
	msg := Message{
		Type:  "gameOver",
		Value: winner,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	c.egress <- data
}
