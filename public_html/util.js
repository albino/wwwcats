// Utility functions

function entities(str) {
	// jQuery hack, I don't know how this works either
	return $("<div/>").text(str).html();
}

function animate(element, property, vInitial, vFinal, incr, unit) {
	// Animate a CSS property

	$(element).removeClass("reveal");
	$(element).css(property, vInitial);
	var vCurrent = vInitial;
	
	var id = setInterval(frame, 15);
	function frame() {
		let hasFinished = false;
		if (vInitial < vFinal) {
			hasFinished = vCurrent >= vFinal;
		} else {
			hasFinished = vCurrent <= vFinal;
		}

		if (hasFinished) {
			clearInterval(id);
			$(element).addClass("reveal");
			$(element).css(property, vInitial.toString() + unit);
			return;
		}

		vCurrent += incr;
		$(element).css(property, vCurrent.toString() + unit);
	}
}

function cardHUD(card, time) {
	$("#card-hud").html("<img class='card' src='assets/card_"+card+".png' />");
	$("#card-hud").removeClass("reveal");
	// The opacity property needs a delay or the animation doesn't work
	// no idea why, better not to ask
	setTimeout(function() {
		$("#card-hud").css("opacity", "1");
	}, 100);

	setTimeout(function() {
		$("#card-hud").css("opacity", "0");

		setTimeout(function() {
			$("#card-hud").empty();
			$("#card-hud").addClass("reveal");
		}, 500);
	}, time);
}
