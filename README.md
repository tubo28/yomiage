# yomiage

読み上げます

## What?

Text-to-speech bot for Discord.

## Features

- `!hi`: Summon bot to voice channel you are joining
- `!bye`: Stop reading
- (todo) `!rand`: Randomize voice to read your text
- (todo) `!lang`: Set language to read your text

## How to run

```sh
export GOOGLE_APPLICATION_CREDENTIALS=credentials.json
export DISCORD_TOKEN=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
go build
./yomiage
```

## Depends on

- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [Google Cloud Text-to-Speech](https://cloud.google.com/text-to-speech)
