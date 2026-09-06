# podcast factory

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white) ![Anthropic](https://img.shields.io/badge/Anthropic-Claude-D97757?logo=anthropic&logoColor=white) ![ElevenLabs](https://img.shields.io/badge/ElevenLabs-TTS-000000?logo=elevenlabs&logoColor=white) ![Mistral](https://img.shields.io/badge/Mistral-Voxtral-FA520F?logo=mistralai&logoColor=white)

Turns an article URL into a narrated podcast episode, because I get carsick reading during my commute to work so I would rather listen to my bookmarks.

A GitHub Action fetches the article, writes a spoken-summary transcript with Claude, synthesizes audio with ElevenLabs or Voxtral, and publishes everything as a [GitHub Release](https://github.com/nadiaenh/podcast-service/releases).

<!--Add DEMO here-->

## Setup

Requires the GitHub CLI (`gh`), authenticated.

```sh
git clone git@github.com:nadiaenh/podcast-service.git
cd podcast-service
./setup.sh
```

`setup.sh` creates `.env` from `.env.example` (fill in your keys, then run it again) and pushes the values to GitHub as repo secrets for the Action. Only accounts with write access to the repo can trigger the Action, so the secrets stay yours even though the repo is public.

## Usage

Trigger a run from the GitHub UI (Actions → "Generate new podcast" → Run workflow), or from the CLI:

```sh
gh workflow run podcast.yml -f url=https://x.ai/news/designing-grok-bot
gh workflow run podcast.yml -f url=https://x.ai/news/designing-grok-bot -f provider=voxtral
```

Each run publishes a Release (e.g. `Episode 1 - Designing Grok Bot`) containing the MP3 (`ep-1-designing-grok-bot.mp3`), the parsed article (`ep-1-designing-grok-bot-source.md`), the transcript (`ep-1-designing-grok-bot-transcript.txt`), and the source code. Re-running the same URL replaces its episode.
