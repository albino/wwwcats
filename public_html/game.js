var GameState = function() {
	// Websocket handle

	this.conn = null;

	// Player and lobby name

	this.name = null;
	this.lobby = null;

	// Assets

	this.images = [];

	this.loadImage = function(img) {
		this.images.push(img);
	}
}

function startGame(gameState) {
	console.log(gameState);
}
