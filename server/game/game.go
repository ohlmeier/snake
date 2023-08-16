package game

import (
	"errors"
	"log"
	"math/rand"
)

const (
	gridSize  = 20
	frameRate = 10
)

type Game struct {
	Players  map[string]Player
	Food     Position
	GridSize int
	Quit     chan bool `json:"-"`
}

type Position struct {
	X, Y int
}

type Player struct {
	ID       string
	Position Position
	Velocity Position
	Snake    []Position
}

func New() *Game {
	game := &Game{
		Players:  make(map[string]Player),
		GridSize: gridSize,
		Quit:     make(chan bool),
	}
	game.randomFood()
	return game
}

func (g *Game) AddPlayerOne(ID string) {
	g.Players[ID] = Player{
		Position: Position{3, 10},
		Velocity: Position{1, 0},
		Snake: []Position{
			{1, 10},
			{2, 10},
			{3, 10},
		},
	}
}

func (g *Game) AddPlayerTwo(ID string) {
	g.Players[ID] = Player{
		Position: Position{18, 10},
		Velocity: Position{0, 0},
		Snake: []Position{
			{20, 10},
			{19, 10},
			{18, 10},
		},
	}
}

func (g *Game) Loop() string {

	for key, player := range g.Players {
		log.Printf("Player %s has vel %v", key, player.Velocity)
		log.Printf("Player %s has pos %v", key, player.Position)
		player.Position.X += player.Velocity.X
		player.Position.Y += player.Velocity.Y
		log.Printf("Player %s has pos %v after apply vel", key, player.Position)
		g.Players[key] = player
	}

	for _, player := range g.Players {
		if player.Position.X < 0 || player.Position.X > g.GridSize ||
			player.Position.Y < 0 || player.Position.Y > g.GridSize {
			return player.ID
		}
	}

	for key, player := range g.Players {
		if g.Food.X == player.Position.X && g.Food.Y == player.Position.Y {
			player.Snake = append(player.Snake, player.Position)
			player.Position.X += player.Velocity.X
			player.Position.Y += player.Velocity.Y
			g.Players[key] = player
			g.randomFood()
		}
	}

	for key, player := range g.Players {
		if player.Velocity.X > 0 || player.Velocity.Y > 0 {
			for _, position := range player.Snake {
				if position.X == player.Position.X && position.Y == player.Position.Y {
					return player.ID
				}
			}

			player.Snake = append(player.Snake, player.Position)
			player.Snake = player.Snake[1:]
			g.Players[key] = player
		}
	}

	//log.Printf("%v\n", g)

	return ""
}

func (g *Game) randomFood() {
	food := Position{
		rand.Intn(g.GridSize),
		rand.Intn(g.GridSize),
	}

	for _, player := range g.Players {
		for _, position := range player.Snake {
			if position.X == food.X && position.Y == food.Y {
				g.randomFood()
				return
			}
		}
	}

	g.Food = food
}

func GetUpdateVelocity(keyCode int) (Position, error) {
	switch keyCode {
	case 37: //left
		return Position{-1, 0}, nil
	case 38: // up
		return Position{0, 1}, nil
	case 39: //right
		return Position{1, 0}, nil
	case 40: //down
		return Position{0, -1}, nil
	case 65: //left
		return Position{-1, 0}, nil
	case 87: // up
		return Position{0, 1}, nil
	case 68: //right
		return Position{1, 0}, nil
	case 83: //down
		return Position{0, -1}, nil
	}

	return Position{}, errors.New("invalid key")
}
