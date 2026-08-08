# podcast factory

## deploy

```sh
git clone git@github.com:nadiaenh/podcast-service.git
cd podcast-service

# bootstrap aws/pulumi/gh + saves API keys to `.env` and to encrypted Pulumi config.
./setup.sh 

# deploy.
pulumi up --refresh
```

## use

```sh
curl -X POST "$(pulumi stack output functionUrl)" \
  -H "x-api-key: $(pulumi config get apiKey)" \
  -H "content-type: application/json" \
  -d '{"url": "https://depot.dev/blog/the-end-of-push-wait-guess-ci"}'
```

```json
{"mp3Url": "https://podcast-service-mp3s-xxxx.s3.amazonaws.com/...", "title": "..."}
```

mp3Url is a presigned S3 link, valid 24h.

## teardown

```sh
pulumi destroy
```
