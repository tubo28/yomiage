# yomiage

読み上げます

## What?

Text-to-speech bot for Discord.

## Commands

| Command                     |                                                                                                            |
| --------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `!hi`                       | Summon the bot. The bot will read out the text of the channel where this command was entered.              |
| `!bye`                      | Stop reading.                                                                                              |
| `!rand`                     | Randomize voice to read your text.                                                                         |
| `!lang`                     | Get the language to read your text.                                                                        |
| `!lang set <language code>` | Set the language to read your text to `<language code>`. See "language selection" section for the details. |
| `!help`                     | Show usage.                                                                                                |

## Language selection

The language code to read each user's text is selected based on the following rules in that order:

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
- SQLite 3
