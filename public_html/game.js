var GameState = function() {
	// Websocket handle

	this.conn = null;

	// Player and lobby name

	this.name = "";
	this.lobby = "";

	// Assets

	this.images = [];

	this.loadImage = function(img) {
		this.images.push(img);
	}

	this.start = function() {
		// Run the game!

		$("#player-name").text(this.name);
		$("#lobby-name").text(this.lobby);

		// Scroll wheel hack - allows the user to scroll the card deck horizontally
		$("#card-deck").mousewheel(function(ev, delta) {
			this.scrollLeft -= (delta * 30);
			ev.preventDefault();
		});

		// Send the contents of the chat box when enter is pressed
		// feat. a stupid scope hack
		(function (gameState) {
			$("#chat-message").keydown( function(ev) {
				if (ev.key === "Enter") {
					if ($( this ).val().startsWith("/")) {
						// Send as a command
						let msg = $( this ).val().substring(1);
						gameState.send(msg);
					} else {
						// Send as a message
						let msg = $( this ).val();
						gameState.send("chat " + msg);
					}

					// Clear the box
					$( this ).val("");
				}
			});
		})(this);
		
		this.console("<span style='color:yellow'>Welcome to Detonating Cats!</span>");

		// We're ready to bring the game board into view
		$("body").css("background-color", "black");
		$("#welcome").toggleClass("reveal");
		$("#game-view").toggleClass("reveal");
	}

	this.console = function(msg) {
		$("#game-log").append(msg+"<br />");

		// Scroll to bottom
		$("#game-log").scrollTop($("#game-log")[0].scrollHeight);
	}

	this.readFromServer = function(ev) {
		var parts = ev.data.split(" ");

		if (parts[0] == "err") {
			alert(strings[parts[1]]);
			this.conn.close();
			this.conn = null;
			return;
		}

		if (parts[0] == "joins" && parts[1] == this.name) {
			// We're in!
			this.start();
			return;
		}

		if (parts[0] == "spectators") {
			// Update spectators list
			$("#spectator-list").empty();

			for (var i=1; i < parts.length; i++) {
				let encoded = entities(parts[i]);
				$("#spectator-list").append("<li>"+encoded+"</li>");
			}

			return;
		}

		if (parts[0] == "players") {
			// Update players list
			$("#player-list").empty();

			for (var i=1; i < parts.length; i++) {
				let encoded = entities(parts[i]);
				$("#player-list").append("<li>"+encoded+"</li>");
			}

			return;
		}

		if (parts[0] == "joins") {
			// Announce in console
			let encoded = entities(parts[1]);
			this.console("<span style='color:green'>"+encoded+" joined as a spectator.</span>");

			// Update spectators list
			$("#spectator-list").append("<li>"+encoded+"</li>");

			return;
		}

		if (parts[0] == "parts") {
			// Announce in console
			let encoded = entities(parts[1]);
			this.console("<span style='color:red'>"+encoded+" left the game.</span>");

			// Update spectators list if appropriate
			$("#spectator-list > li").each( function() {
				if ($( this ).html() == encoded) {
					$( this ).remove();
				}
			} );

			return;
		}

		if (parts[0] == "upgrades") {
			// Announce in console
			let encoded = entities(parts[1]);
			this.console("<span style='color:green'>"+encoded+" is now playing.</span>");

			// Update spectators list
			$("#spectator-list > li").each( function() {
				if ($( this ).html() == encoded) {
					$( this ).remove();
				}
			} );

			// Server will send players list separately, no worries

			return;
		}

		if (parts[0] == "downgrades") {
			// Announce in console
			let encoded = entities(parts[1]);
			this.console("<span style='color:red'>"+encoded+" is now spectating.</span>");

			// Update spectators list
			$("#spectator-list").append("<li>"+encoded+"</li>");

			// Server will send players list separately, no worries

			return;
		}

		if (parts[0] == "chat") {
			let encodedName = entities(parts[1]);
			let encodedMsg = entities(ev.data.substring(parts[1].length + 5));
			this.console(encodedName+": "+encodedMsg);

			return;
		}

		console.warn("WARN: received unknown data from server: "+ev.data);
	}

	this.send = function(msg) {
		this.conn.send(msg);
	}
}
