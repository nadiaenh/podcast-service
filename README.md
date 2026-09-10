<p align="center"> 
    <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25"> <img src="https://img.shields.io/badge/Anthropic-D97757?logo=anthropic&logoColor=white" alt="Claude"> 
    <img src="https://img.shields.io/badge/ElevenLabs-000000?logo=elevenlabs&logoColor=white" alt="ElevenLabs"> 
    <img src="https://img.shields.io/badge/Voxtral-FA520F?logo=mistralai&logoColor=fff" alt="Voxtral"> 
    <a href=".github/workflows/checks.yml"><img src="https://github.com/nadiaenh/sockpuppet/actions/workflows/checks.yml/badge.svg" alt="Checks"></a> 
</p>

**sockpuppet** is a pipeline that turns written content into a narrated MP3. See recent episodes in the [Releases](https://github.com/nadiaenh/sockpuppet/releases) page. Each release contains the source content as-ingested, the narrated transcript, and the MP3.

<p align="center"><img width="250" src="https://media.tenor.com/3ALrVOdGNYoAAAAi/happy-funny.gif" alt="Sock puppet"></p>

## Setup

```sh
git clone git@github.com:nadiaenh/sockpuppet.git
cd sockpuppet
gh auth login
./setup.sh
```

## Usage

Trigger a run from the GitHub UI (Actions → "Generate new podcast" → Run workflow), or from the CLI:

```sh
gh workflow run podcast.yml -f url=https://x.ai/news/designing-grok-bot
gh workflow run podcast.yml -f url=https://x.ai/news/designing-grok-bot -f provider=elevenlabs
```

Run the pipeline locally against your `.env` keys:

```sh
mkdir -p out
go run . verify-keys
go run . fetch https://x.ai/news/designing-grok-bot out
go run . transcript https://x.ai/news/designing-grok-bot out
go run . speak out
go test ./...
```

## Demo

![Four CLI commands producing an episode Release](assets/demo.svg)
