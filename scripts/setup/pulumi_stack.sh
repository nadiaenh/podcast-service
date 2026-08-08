#!/usr/bin/env bash

passphrase_file=".pulumi-passphrase"
if [ -f "$passphrase_file" ]; then
  ok "pulumi passphrase already exists"
else
  passphrase=$(openssl rand -base64 32)
  echo "$passphrase" > "$passphrase_file"
  chmod 600 "$passphrase_file"
  ok "generated pulumi passphrase, saved to gitignored file $passphrase_file"
  echo "  → $passphrase"
fi
export PULUMI_CONFIG_PASSPHRASE
PULUMI_CONFIG_PASSPHRASE=$(cat "$passphrase_file")

quiet bash -c "cd infra && go mod download" || fail "infra go mod download failed" "see error output below"
ok "infra dependencies downloaded"
quiet bash -c "cd lambda && go mod download" || fail "lambda go mod download failed" "see error output below"
ok "lambda dependencies downloaded"

quiet pulumi login "s3://${bucket}?region=${region}" || fail "pulumi login failed" "see error output below"
ok "pulumi logged into s3 backend"

if pulumi stack select "$stack" >"$log" 2>&1; then
  ok "pulumi stack '$stack' selected"
else
  quiet pulumi stack init "$stack" || fail "failed to init pulumi stack" "see error output below"
  ok "pulumi stack '$stack' created"
fi

if quiet pulumi config set --secret _probe ok && quiet pulumi config rm _probe; then
  ok "pulumi passphrase matches stack encryption salt"
else
  echo "→ passphrase does not match existing encryption salt in Pulumi.$stack.yaml — resetting salt"
  grep -vE '^encryptionsalt:' "Pulumi.$stack.yaml" > "Pulumi.$stack.yaml.tmp" && mv "Pulumi.$stack.yaml.tmp" "Pulumi.$stack.yaml"
  quiet pulumi config rm apiKey || true
  quiet pulumi config set --secret _probe ok && quiet pulumi config rm _probe \
    || fail "could not re-initialize stack encryption" "see error output below"
  ok "stack encryption salt reset with your passphrase"
fi

quiet pulumi config set aws:region "$region" || fail "failed to set region" "see error output below"
ok "region set to $region"

if pulumi config get apiKey >/dev/null 2>&1; then
  ok "api key already set"
else
  quiet pulumi config set --secret apiKey "$(openssl rand -hex 24)" || fail "failed to set api key" "see error output below"
  ok "generated api key"
fi
api_key=$(pulumi config get apiKey)
