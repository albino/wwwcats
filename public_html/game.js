var GameState = function() {
	// Websocket handle

	this.conn = null;

	this.started = false;
	this.defusing = false;
	this.ourTurn = false;
	this.locked = false;
	this.favouring = false;
	this.combo = 1;

	// Player and lobby name

	this.name = "";
	this.lobby = "";

	this.nowPlaying = "";
	this.players = [];

	// Assets

	this.assets = {};

	this.loadAsset = function(url, el) {
		this.assets[url] = el;
	}

	this.resetButtons = function() {
		let buttons = $(".combo-btn").toArray();
		buttons.forEach(function(btn) {
			$(btn).removeClass("active");
		});
		this.combo = 1;
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

		// 2x and 3x buttons
		(function(gameState) {
			let buttons = $(".combo-btn").toArray();
			buttons.forEach(function(btn) {
				$(btn).on("click", function() {
					// deactive all the buttons, except this one
					buttons.forEach(function(b) {
						if (b != btn) {
							$(b).removeClass("active");
						}
					});
					$(this).toggleClass("active");

					if ($(this).hasClass("active")) {
						gameState.combo = $(this).attr("id") == "2x-button" ? 2 : 3;
					} else {
						gameState.combo = 0;
					}
				});
			});
		})(this);

		// Sort button
		(function(gameState) {
			$("#sort-button").on("click", function() {
				gameState.send("sort");
			});
		})(this);

		// Mute button
		(function(gameState) {
			$("#mute-button").on("click", function() {
				if (gameState.assets["atomic.ogg"].muted) {
					gameState.assets["atomic.ogg"].muted = false;
					$("#mute-button").text("Mute sound");
				} else {
					gameState.assets["atomic.ogg"].muted = true;
					$("#mute-button").text("Unmute sound");
				}
			});
		})(this);

		(function(gameState) {
			window.onbeforeunload = function () {
				if (gameState.conn.readyState !== WebSocket.CLOSED) {
					gameState.conn.onclose = null;
					return ""; // Display confirmation message
				}
			};
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

	this.drawPlayerList = function() {
		$("#player-list").empty();
		this.players.forEach(x => $("#player-list").append("<li>"+x+"</li>"));

		(function(gameState) {
			$("#player-list > li").each( function() {
				if ($( this ).html() == gameState.nowPlaying) {
					$( this ).append("<span id='now-playing-mark' style='color:red'> *</span>");
				}
			} );
		})(this);
	}

	this.readFromServer = function(ev) {
		var parts = ev.data.split(" ");

		// TODO: cleanup and refactor

		if (parts[0] == "err") {
			alert(strings[parts[1]]);
			this.conn.close();
			this.conn = null;
			return;
		}
		if (parts[0] == "version" && parts[1] != REVISION) {
			alert(strings["bad_version"]);
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
			this.players = parts.slice(1).map(x => entities(x));
			this.drawPlayerList();

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
						if (gameState.locked) {
							return;
						}

						if (gameState.favouring) {
							// Favour NOPE-logic is handled server-side :)
							gameState.send("a favour_what "+cardNo.toString());
							gameState.favouring = false;
							return;
						}

						if (gameState.combo > 1) {
							// Do we have enough cards?
							let cards = 0;
							for (var j=1; j < parts.length; j++) {
								if (parts[j] == cardName) {
									cards++;
									if (cards == gameState.combo) {
										break;
									}
								}
							}
							if (cards < gameState.combo) {
								gameState.console("You don't have enough " + strings["card_"+cardName] +
									" cards to do that!");
								return;
							}

							if (gameState.ourTurn) {
								gameState.send("play_multiple "+gameState.combo.toString()+" "+
									cardName);
								gameState.resetButtons();
								return;
							}
						}

						if ((gameState.ourTurn || cardName === "nope") && !cardName.startsWith("random")
								&& ( (!gameState.defusing && cardName !== "defuse")
								||    (gameState.defusing && cardName === "defuse") )) {
							gameState.send("play "+cardNo.toString());
							gameState.defusing = false;
						}
					});
				})(this, i-1, parts[i]);

				$("#card-deck").append(card);
			}

			return;
		}

		if (ev.data == "draw_pile yes") {
			$("#draw-pile").html("<img class='card' src='assets/card_back.png' />");
			$("#draw-pile-counter").removeClass("reveal");
			return;
		}
		if (ev.data == "draw_pile no") {
			$("#draw-pile").html("");
			$("#draw-pile-counter").addClass("reveal");
			return;
		}
		if (parts[0] == "cards_left") {
			$("#remaining-card-count").text(parts[1]);
			return;
		}

		if (parts[0] == "now_playing") {
			this.nowPlaying = entities(parts[1]);
			this.console("<span style='color:yellow'>It is "+this.nowPlaying+"'s turn.</span>");

			this.drawPlayerList();

			// cmp with raw data because name isn't stored encoded
			if (parts[1] == this.name) {
				this.ourTurn = true;
				this.assets["atomic.ogg"].play();
				document.title = strings["title_alert"];
			} else {
				this.ourTurn = false;
				document.title = strings["title_normal"]; // a bit fragile, needs to be same as <title>
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
			this.nowPlaying = "";
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

		if (parts[0] == "played_multiple") {
			let encoded = entities(parts[1]);
			this.console(encoded+" played "+parts[2]+"x "+strings["card_"+parts[3]]+".");

			$("#discard-pile").html("<img class='card' src='assets/card_"+parts[3]+".png' />");

			if (parts[2] == 2) {
				cardHUD3([parts[3], parts[3]], 1000);
			} else {
				cardHUD3([parts[3], parts[3], parts[3]], 1000);
			}

			return;
		}

		if (parts[0] == "no_discard") {
			$("#discard-pile").html("");
			return;
		}

		if (parts[0] == "q") {
			if (parts[1] == "favour_what") {
				this.favouring = true;
				let perpetrator = entities(parts[2]);
				this.console("<span style='color:deepskyblue'>" + perpetrator +
					" is asking you for a favour.</span>");
			} else if (parts[1] == "favour_who" || parts[1] == "random_who" || parts[1] == "steal_who") {
				(function (gameState) {
					modalChoice(function(player) {
						gameState.send("a "+parts[1]+" "+player);
					}, strings["question_"+parts[1]], gameState.players, entities(gameState.name), null);
				})(this);
			} else if (parts[1] == "steal_what") {
				(function (gameState) {
					modalChoice(function(card) {
						gameState.send("a "+parts[1]+" "+card);
					}, strings["question_"+parts[1]], cards, null, x => strings["card_"+x]);
				})(this);
			} else {
				ans = prompt(strings["question_"+parts[1]]);
				this.send("a "+parts[1]+" "+ans);
			}

			return;
		}
		if (parts[0] == "q_cancel") {
			this.favouring = false;
		}

		if (parts[0] == "seen") {
			cardHUD3(parts.slice(1), 2000);
			this.console("You saw "+strings["card_"+parts[1]]+", "+strings["card_"+parts[2]]+" and "+
				strings["card_"+parts[3]]+".");
			return;
		}

		if (parts[0] == "favoured" || parts[0] == "favour_complete") {
			let perpetrator = entities(parts[1]);
			let victim = entities(parts[2]);
			if (parts[0] == "favoured") {
				this.console(perpetrator+" is asking "+victim+" for a favour.");
			} else {
				this.console(victim+" gave "+perpetrator+" a favour.");
			}
			return;
		}
		if (parts[0] == "favour_recv" || parts[0] == "favour_gave") {
			let remotePlayer = entities(parts[1]);
			let card = strings["card_"+parts[2]];
			if (parts[0] == "favour_recv") {
				this.console(remotePlayer + " gave you <span style='color:orange'>" + card + "</span>.");
				cardHUD(parts[2], 2000);
			} else {
				this.console("You gave " + remotePlayer + " <span style='color:orange'>" + card + "</span>.");
			}
			return;
		}

		if (parts[0] == "randomed" || parts[0] == "random_n") {
			let perpetrator = entities(parts[1]);
			let victim = entities(parts[2]);
			if (parts[0] == "randomed") {
				this.console(perpetrator+" took a random card from "+victim+".");
			} else {
				this.console(perpetrator+" asked "+victim+
					" for a random card, but they had nothing to give away!");
			}

			return;
		}
		if (parts[0] == "random_recv" || parts[0] == "random_gave") {
			let remotePlayer = entities(parts[1]);
			let card = strings["card_"+parts[2]];
			if (parts[0] == "random_recv") {
				this.console("You randomly took <span style='color:orange'>"+card+"</span> from "+
					remotePlayer+".");
			} else {
				this.console(remotePlayer+" randomly took <span style='color:orange'>"+card+
					"</span> from you.");
			}
			cardHUD(parts[2], 2000);
			return;
		}

		if (parts[0] == "steal_n" || parts[0] == "steal_y") {
			let perpetrator = entities(parts[1]);
			let victim = entities(parts[2]);
			if (parts[0] == "steal_y") {
				this.console(perpetrator+" stole <span style='color:orange'>"+
					strings["card_"+parts[3]]+"</span> from "+victim+"!");
				cardHUD(parts[3], 2000);
			} else {
				this.console(perpetrator+" asked "+victim+" for <span style='color:orange'>"
					+strings["card_"+parts[3]]+"</span>, but ended up empty-handed!");
			}
			return;
		}

		if (parts[0] == "lock") {
			this.locked = true;
			return;
		}
		if (parts[0] == "unlock") {
			this.locked = false;
			return;
		}

		console.warn("WARN: received unknown data from server: "+ev.data);
	}

	this.send = function(msg) {
		this.conn.send(msg);
	}
}
