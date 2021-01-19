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

function cardHUD3(cards, time) {
	cards.forEach(function(card) {
		$("#card-hud-3").append("<img class='card' src='assets/card_"+card+".png' />");
	});
	$("#card-hud-3-wrapper").removeClass("reveal");

	setTimeout(function() {
		$("#card-hud-3-wrapper").css("opacity", "1");
	}, 100);

	setTimeout(function() {
		$("#card-hud-3-wrapper").css("opacity", "0");

		setTimeout(function() {
			$("#card-hud-3").empty();
			$("#card-hud-3-wrapper").addClass("reveal");
		}, 500);
	}, time);
}

function modalChoice(cb, question, options, exclude, transform) {
	// NO anti-xss
	
	let modalHud = document.createElement("DIV");
	$(modalHud).addClass("modal-hud");
	let modal = document.createElement("DIV");
	$(modal).addClass("modal");
	$(modal).html("<span>"+question+"</span>");

	let modalOptions = document.createElement("UL");
	$(modalOptions).addClass("modal-options");

	options.forEach(function (opt) {
		if (opt === exclude) {
			return;
		}

		let button = document.createElement("LI");

		if (transform) {
			button.innerHTML = transform(opt);
		} else {
			button.innerHTML = opt;
		}

		button.addEventListener("click", function() {
			cb(opt);

			$(modalHud).css("opacity", "0");
			setTimeout(function() {
				document.getElementById("modal-container").removeChild(modalHud);
			}, 500);
		}, false);
		modalOptions.appendChild(button);
	});

	modal.appendChild(modalOptions);
	modalHud.appendChild(modal);
	document.getElementById("modal-container").appendChild(modalHud);

	setTimeout(function() {
		$(modalHud).css("opacity", "1");
	}, 100);
}
