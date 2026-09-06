# podcast factory

Turns an article URL into a narrated MP3. Fetches the article, writes a script with Claude, and synthesizes audio with ElevenLabs or Voxtral. Run it locally, or trigger a GitHub Action that publishes the MP3 as a GitHub Release.

## 1. One-time setup

Requires Go 1.25+ (and `gh`, authenticated, for the Action).

```sh
git clone git@github.com:nadiaenh/podcast-service.git
cd podcast-service
./setup.sh
```

`setup.sh` creates `.env` from `.env.example` and, if you want, pushes those values to GitHub as repo secrets for the Action.

`.env` / secrets:

| Variable | Required | Notes |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | yes | console.anthropic.com |
| `ELEVENLABS_API_KEY` / `ELEVENLABS_VOICE_ID` | for elevenlabs | elevenlabs.io |
| `VOXTRAL_API_KEY` / `VOXTRAL_VOICE_ID` | for voxtral | console.mistral.ai |
| `TTS_PROVIDER` | no | `elevenlabs` (default) or `voxtral` |

Only accounts with write access to the repo can trigger the Action, so the secrets stay yours even though the repo is public.

## 2. Usage

When triggered locally, it writes `./podcast.mp3` (or the path you pass):

```sh
go run . https://example.com/some-article
go run . https://example.com/some-article out.mp3
```

When triggered via the GitHub Action — triggers the workflow, waits for it, prints the MP3's download URL:

```sh
./scripts/make.sh https://example.com/some-article
./scripts/make.sh https://example.com/some-article voxtral
```

Each run publishes a Release tagged `ep-<hash of url>`; re-running the same URL replaces it.

## 3. Demo

_TODO: add a screen recording or screenshots of a run._
