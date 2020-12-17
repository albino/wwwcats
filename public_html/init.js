(function () {

	// Everything that runs as soon as the page loads; sets up the page and handles the
	// connection to the server.

	var gameState = new GameState();

	$( document ).ready(function () {
		// Set up the game
		
		// Load assets
		// TODO: audio assets
		var imageAssets = ["wood.jpg"];
		var promises = [];
		var assetsLoaded = 0;

		for (var i = 0; i < imageAssets.length; i++) {
			(function(url, promise) {
				var img =  new Image();
				img.onload = function() {
					// Increment the loading counter to convince the user we're doing something
					assetsLoaded += 1;
					document.getElementById("loading-assets").innerHTML = assetsLoaded;

					promise.resolve();
				};
				img.src = "assets/"+url;

				gameState.loadImage(img);
			})(imageAssets[i], promises[i] = $.Deferred());
		}

		// Once all the promises have resolved (all assets loaded), call the function to 
		// display the welcome page.
		$.when.apply($, promises).done(welcomePage);
	});

	function welcomePage() {
		// Transition loading screen -> welcome page

		console.log("Assets loaded successfully");

		$("#welcome-join").bind("click touchstart", joinGame);

		$("#loading").css("display", "none");
		$("#welcome").css("display", "block");
	}

	function joinGame() {
		// Communicate with the server, make sure we join the game ok, render the board

		if (gameState.conn != null) {
			alert(strings["already_connecting"]);
			return;
		}
		
		var lobby = $("#welcome-lobby").val() || $("#welcome-lobby").attr("placeholder");
		var user  = $("#welcome-username").val() || $("#welcome-username").attr("placeholder");

		if (user.includes(" ")) {
			alert(strings["one_word"]);
			return;
		}

		gameState.conn = new WebSocket("ws://" + location.host + "/ws");

		gameState.conn.onopen = function () {
			gameState.conn.send("join_lobby " + lobby + " " + user);
		}

		gameState.conn.onmessage = function (e) {
			var parts = e.data.split(" ");

			if (parts[0] == "err") {
				alert (strings[parts[1]]);
				gameState.conn.close();
				gameState.conn = null;
				return;
			}

			if (parts[0] == "joins" && parts[1] == user) {
				// We're in!

				gameState.name = user;
				gameState.lobby = lobby;

				startGame(gameState);

				return;
			}

			console.warn("WARN: received unknown data from server: "+e.data);
		}
	}
})();
