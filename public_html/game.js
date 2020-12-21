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

	// Run the game!

	this.start = function() {
		$("#player-name").text(this.name);
		$("#lobby-name").text(this.lobby);

		// Scroll wheel hack - allows the user to scroll the card deck horizontally
		$("#card-deck").mousewheel(function(ev, delta) {
			this.scrollLeft -= (delta * 30);
			ev.preventDefault();
		});

		// We're ready to bring the game board into view

		$("#welcome").toggleClass("reveal");
		$("#game-view").toggleClass("reveal");
	}
}
