# yomiage

読み上げます

## What?

Text-to-speech bot for Discord.

## Features

- `!hi`: Summon the bot. Reads the text channel in which this command is send on the voice channel you are joining.
- `!bye`: Stop reading.
- `!rand`: Randomize voice to read your text.
- `!lang` or `!lang get`: Get the language to reads your text.
- `!lang set <language code>`: Set the language to reads your text to `<language code>`. See language selection section for details of `!lang` command.

## Language selection

The "language code" to read each user's text is selected based on the following rules in that order:

1. `DEFAULT_TTS_LANG` environment variable if set.
2. Language set by `!lang` command if set.
3. `en-US` (US English).

Examples of valid language codes are:

- `en` (any English)
- `en-US` (US English)
- `cmn` (Chinese)
- `cmn-CN` (Chine Chinese)
- `ja`, `ja-JP` (Japanese)

For all available codes, see "Language code" column of [Supported voices and languages](https://cloud.google.com/text-to-speech/docs/voices).

This is passed to `language_code` parameter of [VoiceSelectionParams](https://cloud.google.com/text-to-speech/docs/reference/rpc/google.cloud.texttospeech.v1#voiceselectionparams).

## How to run

```sh
go build
export GOOGLE_APPLICATION_CREDENTIALS=credentials.json
export DISCORD_TOKEN=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
export DEFAULT_TTS_LANG=en-US
./yomiage
```

## Depends on

- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [Google Cloud Text-to-Speech](https://cloud.google.com/text-to-speech)
