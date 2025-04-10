# movementpossession

simple app to record and translate wasd movements into chat commands for noit.ing

build using `go build` (you need go toolchain)

ensure twitch gets connected before doing anything

resize window on top of the stream to allow for clicks to do accurate mouse position sets

## keybinds

WASD and spacebar are used to record the movement sequence

once you started pressing the key you must release it or change your inputs state/combination in order
for the chat command to be rendered and have duration

send key sends sequence to twitch

clear clears recorded sequence (to start a new one, inital delay before inputs ignored)

send+clear used to quickly chain-send recordings

minus key enables auto-send that is very scuffed and would send current recorded inputs every 5 seconds

## config

keys can be populated from key names available in keys.go

change your username in config.json, leave redirect and client id as is if you are not making your own application

keep in mind that your USER TOKEN WILL BE STORED IN PLAIN TEXT IN TOKEN.TXT!!!

