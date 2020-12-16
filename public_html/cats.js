$( document ).ready(function () {
	var conn = new WebSocket("ws://" + location.host + "/ws");
	conn.onopen = function () {
		console.log("Connected");
		conn.send("ping!");
	}
	conn.onmessage = function (e) {
		console.log(e.data);
	}
});
