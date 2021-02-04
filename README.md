# Detonating Cats

A web-based clone of the popular _[Exploding Kittens](https://explodingkittens.com)_ card game. I wrote this
because I wanted to play the game online with my friends during the December 2020 coronavirus lockdown and
was dissatisfied with the existing clones of the game.

## Features

This is a pretty much feature-complete clone of the base game. The objective is to permit any play
which would be allowed by the rules in a real-life game.

I have done my best to emulate the most important aspects of the game faithfully; while no computer
card game is ever 100% true to the experience of the real thing, this version attempts to preserve
many of the features of _Exploding Kittens_ which make the game dramatic, dynamic and fun.

The game is fast and lightweight, and can be hosted on any server with no dependencies; all you need
is the game server binary and the corresponding static client bundle.

## Caveats

  * I designed everything myself, including drawing the cards. I have no idea about user interface design,
  so the whole thing feels a bit early-2000s.
  * The game has not been well-tested. I made this for fun, and there are probably bugs. You've been warned!
  * If you're interested in improving any aspect of the game, contributions are very welcome. Please open an
  issue or pull request on GitHub!

## Compilation

To compile, you need a recent version of [Go](https://golang.org) – version 1.14 or newer is probably fine.
All the other dependencies are fetched automatically.

After cloning the repository:  
```
$ go build
$ # Or, for a statically linked, stripped binary:
$ CGO_ENABLED=0 go build -ldflags "-s -w"
```

Just run the `wwwcats` binary to start the server, and `./wwwcats -h` for help with more options.

## License

Copyright (C) 2021 Lawrence Brown.

"Exploding Kittens" is a trademark of Exploding Kittens Inc.

The program's source code is released under the terms of the GNU Affero General Public License (version 3).
Please see the `LICENSE` file for more information.
