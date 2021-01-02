var GameState = function() {
	// Websocket handle

	this.conn = null;

	this.started = false;
	this.defusing = false;
	this.ourTurn = false;

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
		// Doesn't actually _start_ the game, but rather starts the game client

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

		// Draw a card
		(function(gameState) {
			$("#draw-pile").on('click', function() {
				if (gameState.ourTurn) {
					gameState.send("draw");
				}
			});
		})(this);

		this.console("<span style='color:yellow'>Welcome to Detonating Cats!</span>");

		// We're ready to bring the game board into view
		$("body").css("background-color", "black");
		$("#welcome").toggleClass("reveal");
		$("#game-view").toggleClass("reveal");

		this.started = true;
	}

	this.console = function(msg) {
		$("#game-log").append(msg+"<br />");

		// Scroll to bottom
		$("#game-log").scrollTop($("#game-log")[0].scrollHeight);
	}

	this.readFromServer = function(ev) {
		console.log("<< "+ev.data);

		var parts = ev.data.split(" ");

		if (parts[0] == "err") {
			alert(strings[parts[1]]);
			this.conn.close();
			this.conn = null;
			return;
		}

		if (parts[0] == "joins" && parts[1] == this.name) {
			// We're in!
			if (!this.started) {
				this.start();
			}
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

		if (parts[0] == "message") {
			// This refers to the 'message' container in the middle of the board
			// that can be used to display useful game info
			let msg = strings["message_"+ev.data.substring(8)];
			$("#message").html(msg);
			$("#message-container").removeClass("reveal");

			return;
		}

		if (parts[0] == "clear_message") {
			$("#message").html("");
			$("#message-container").addClass("reveal");

			return;
		}

		if (parts[0] == "bcast") {
			let msg = strings["bcast_"+ev.data.substring(6)];
			this.console(msg);

			return;
		}

		if (parts[0] == "hand") {
			$("#card-deck").empty();

			for (var i=1; i < parts.length; i++) {
				var card = $("<img class='card' src='assets/card_"+parts[i]+".png' />");

				(function (gameState, cardNo, cardName) {
					card.on("click", function() {
						// TODO: check whether it's ok to play this card or if we are throwing it away
						if (gameState.ourTurn || cardName === "nope") {
							gameState.send("play "+cardNo.toString());
						}
					});
				})(this, i-1, parts[i]);

				$("#card-deck").append(card);
			}

			return;
		}

		if (ev.data == "draw_pile yes") {
			$("#draw-pile").html("<img class='card' src='assets/card_back.png' />");
			return;
		}
		if (ev.data == "draw_pile no") {
			$("#draw-pile").html("");
			return;
		}

		if (parts[0] == "now_playing") {
			let encoded = entities(parts[1]);
			this.console("<span style='color:yellow'>It is "+encoded+"'s turn.</span>");

			$("#now-playing-mark").remove();

			$("#player-list > li").each( function() {
				if ($( this ).html() == encoded) {
					$( this ).append("<span id='now-playing-mark' style='color:red'> *</span>");
				}
			} );

			if (parts[1] == this.name) {
				this.ourTurn = true;
			} else {
				this.ourTurn = false;
			}

			return;
		}

		if (parts[0] == "drew") {
			this.console("You drew <span style='color:orange'>"+strings["card_"+parts[1]]+".</span>");

			// Animation
			cardHUD(parts[1], 2000);

			return;
		}
		if (parts[0] == "drew_other") {
			let encoded = entities(parts[1]);
			this.console("<span style='color:#ccc'>"+encoded+" drew a card.</span>");

			// Animation
			$("#draw-pile-animation").html("<img src='assets/card_back.png' class='card' />");
			animate("#draw-pile-animation", "right", 125, -150, -15, "px");

			return;
		}

		if (parts[0] == "exploded") {
			let encoded = entities(parts[1]);
			this.console("<span style='color:purple'>"+encoded+" drew a Detonating Cat!</span>");

			cardHUD("exploding", 1000);

			return;
		}

		if (parts[0] == "wins") {
			this.ourTurn = false;
			let encoded = entities(parts[1]);
			this.console("<span style='color:deepskyblue'>"+encoded+" won!</span>");
			return;
		}

		if (parts[0] == "defusing") {
			this.console(strings["must_defuse"]);
			this.defusing = true;
			return;
		}

		if (parts[0] == "played") {
			let encoded = entities(parts[1]);
			this.console(encoded+" played "+strings["card_"+parts[2]]+".");
			
			$("#discard-pile").html("<img class='card' src='assets/card_"+parts[2]+".png' />");

			if (parts[2] != "see3") {
				cardHUD(parts[2], 1000);
			}

			return;
		}

		if (parts[0] == "no_discard") {
			$("#discard-pile").html("");
			return;
		}

		if (parts[0] == "q") {
			// TODO: nicer GUI for this
			ans = prompt(strings["question_"+parts[1]]);
			this.send("a "+parts[1]+" "+ans);

			return;
		}

		if (parts[0] == "seen") {
			cardHUD3(parts.slice(1), 2000);
			this.console("You saw "+strings["card_"+parts[1]]+", "+strings["card_"+parts[2]]+" and "+
				strings["card_"+parts[3]]+".");
			return;
		}

		console.warn("WARN: received unknown data from server: "+ev.data);
	}

	this.send = function(msg) {
		console.log(">> "+msg);
		this.conn.send(msg);
	}
}
