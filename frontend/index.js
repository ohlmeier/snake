const BG_COLOUR = '#231f20';
const SNAKE_COLOUR = '#c2c2c2';
const FOOD_COLOUR = '#e66916';

const socket = new WebSocket("ws://localhost:8080/ws")


const generateRandomString = (length) => {
  let result = '';
  const characters =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  const charactersLength = characters.length;
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
};

const clientId = generateRandomString(5)

socket.onopen = () => {
  console.log("Successfully Connected");
};

socket.onclose = event => {
  console.log("Socket Closed Connection: ", event);
  socket.send("Client Closed!")
};

socket.onerror = error => {
  console.log("Socket Error: ", error);
};

socket.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  switch (msg.type) {
    case "gameState":
      handleGameState(msg.gameState)
      break;
    case "gameOver":
      handleGameOver(msg)
      break;
    case "gameCode":
      handleGameCode(msg.value)
      break;
    case "unknownCode":
      handleUnknownCode(msg.value)
      break;
    case "tooManyPlayers":
      handleTooManyPlayers(msg.value)
      break;
  }
};

const gameScreen = document.getElementById('gameScreen');
const initialScreen = document.getElementById('initialScreen');
const newGameBtn = document.getElementById('newGameButton');
const joinGameBtn = document.getElementById('joinGameButton');
const gameCodeInput = document.getElementById('gameCodeInput');
const gameCodeDisplay = document.getElementById('gameCodeDisplay');

newGameBtn.addEventListener('click', newGame);
joinGameBtn.addEventListener('click', joinGame);


function newGame() {
  const msg = {
    type:"newGame",
    client:clientId
  }
  socket.send(JSON.stringify(msg))
  init();
}

function joinGame() {
  const code = gameCodeInput.value;
  const msg = {
    type:"joinGame",
    value:code,
    client:clientId
  }
  socket.send(JSON.stringify(msg))
  init();
}

let canvas, ctx;
let playerNumber;
let gameActive = false;

function init() {
  initialScreen.style.display = "none";
  gameScreen.style.display = "block";

  canvas = document.getElementById('canvas');
  ctx = canvas.getContext('2d');

  canvas.width = canvas.height = 600;

  ctx.fillStyle = BG_COLOUR;
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  document.addEventListener('keydown', keydown);
  gameActive = true;
}

function keydown(e) {
  console.log(e.keyCode)
  const msg = {
    type:"keydown",
    client:clientId,
    key: e.keyCode
  }
  socket.send(JSON.stringify(msg));
}

function paintGame(state) {
  ctx.fillStyle = BG_COLOUR;
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  const food = state.Food;
  const gridsize = state.GridSize;
  const size = canvas.width / gridsize;

  ctx.fillStyle = FOOD_COLOUR;
  ctx.fillRect(food.X * size, food.Y * size, size, size);
  let color = SNAKE_COLOUR

  for (const [playerID, player] of Object.entries(state.Players)) {
    paintPlayer(player, size, color);
    color = 'red'
    }
}

function paintPlayer(playerState, size, colour) {
  console.log(playerState)
  const snake = playerState.Snake;

  ctx.fillStyle = colour;
  for (let cell of snake) {
    ctx.fillRect(cell.X * size, cell.Y * size, size, size);
  }
}

function handleGameState(gameState) {
  console.log("HandleGameState")
  if (!gameActive) {
    return;
  }
  //gameState = JSON.parse(gameState);
  requestAnimationFrame(() => paintGame(gameState));
}

function handleGameOver(data) {
  if (!gameActive) {
    return;
  }
  data = JSON.parse(data);

  gameActive = false;

  if (data.Client === clientId) {
    alert('You Win!');
  } else {
    alert('You Lose :(');
  }
}

function handleGameCode(gameCode) {
  gameCodeDisplay.innerText = gameCode;
}

function handleUnknownCode() {
  reset();
  alert('Unknown Game Code')
}

function handleTooManyPlayers() {
  reset();
  alert('This game is already in progress');
}

function reset() {
  playerNumber = null;
  gameCodeInput.value = '';
  initialScreen.style.display = "block";
  gameScreen.style.display = "none";
}
