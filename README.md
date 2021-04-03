# yomiage

読み上げます

## What?

Text-to-speech bot for Discord.

## Commands

Except `!hi`, `!bye` and `<@bot> help` are WIP.

| Command                       |                                                                                                                  |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `!hi`                         | Summon the bot. The bot will read out the text of the channel where this command was entered.                    |
| `!bye`                        | Stop reading.                                                                                                    |
| `<@bot> help`                 | Show usage.                                                                                                      |
| `<@bot> lang`                 | (WIP) Get the language to read your text.                                                                        |
| `<@bot> lang <language code>` | (WIP) Set the language to read your text to `<language code>`. See "language selection" section for the details. |
| `<@bot> rand`                 | (WIP) Randomize voice to read your text.                                                                         |

## Language selection

The language code to read text is selected based on the following rules in that order:

1. Language set by `!lang` command if set.
1. `DEFAULT_TTS_LANG` environment variable if set.
1. `en-US` (US English).

Examples of valid language codes are:

- `en` (English of not specific region)
- `en-US` (US English)
- `cmn` (Chinese of not specific region)
- `cmn-CN` (Chine Chinese)
- `ja`, `ja-JP` (Japanese)

For all available codes, see "Language code" column of [Supported voices and languages](https://cloud.google.com/text-to-speech/docs/voices).

This is passed to Google TTS API as `language_code` parameter in [VoiceSelectionParams](https://cloud.google.com/text-to-speech/docs/reference/rpc/google.cloud.texttospeech.v1#voiceselectionparams).

## How to run

```sh
go build
export GOOGLE_APPLICATION_CREDENTIALS=credentials.json
export DEFAULT_TTS_LANG=en-US
export DISCORD_TOKEN=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
./yomiage
```

## Deploy with Docker

1. Write Discord token to `secret.env` like `secret.env.sample`
1. `docker-compose up`

## Depends on

- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [Google Cloud Text-to-Speech](https://cloud.google.com/text-to-speech)
- SQLite 3
