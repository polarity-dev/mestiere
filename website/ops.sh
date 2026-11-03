#!/usr/bin/env bash
# Operations for the mestiere.io website.
#
# Hosting: polarity-tools account (404420994311), stack `mestiere-website` in
# eu-west-1, certificate in us-east-1 (required by CloudFront).
# DNS: public zone mestiere.io in the main Polarity account (477398036046),
# where the domain is registered.
#
# Credentials: Shonin profiles. If a command fails with ExpiredTokenException,
# sign in again with `shonin login polarity`.
#
# Usage:
#   website/ops.sh cert      request the ACM certificate and validate it through DNS
#   website/ops.sh stack     create/update the CloudFormation stack
#   website/ops.sh dns       point mestiere.io and www.mestiere.io at the distribution
#   website/ops.sh deploy    build website/dist, upload it to S3 and invalidate CloudFront
#   website/ops.sh outputs   print the stack outputs
#
# First setup, in order: cert, stack, dns, deploy. Afterwards only deploy.
set -euo pipefail

STACK_NAME=mestiere-website
REGION=eu-west-1
DOMAIN=mestiere.io
# SITE_PROFILE= (empty) uses the credentials already in the environment: that
# is how the GitHub workflow runs `deploy` with its OIDC role.
SITE_PROFILE="${SITE_PROFILE-polarity-polarity-tools-admin}"
DNS_PROFILE=sf
DNS_ZONE_ID=Z08888601DMMX4BM3SPNF
CLOUDFRONT_ZONE_ID=Z2FDTNDATAQYW2
WEBSITE_DIR="$(cd "$(dirname "$0")" && pwd)"
DIST_DIR="$WEBSITE_DIR/dist"

site_auth=()
if [[ -n "$SITE_PROFILE" ]]; then
  # Static env credentials would win over the profile: drop them.
  unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE
  site_auth=(--profile "$SITE_PROFILE")
fi

# ${a[@]+...}: an empty array is "unbound" under set -u in bash 3.2 (macOS).
site() { aws ${site_auth[@]+"${site_auth[@]}"} --region "$REGION" "$@"; }
site_us() { aws ${site_auth[@]+"${site_auth[@]}"} --region us-east-1 "$@"; }
dns() { aws --profile "$DNS_PROFILE" "$@"; }

stack_output() {
  site cloudformation describe-stacks --stack-name "$STACK_NAME" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue" --output text
}

# Newest ISSUED (or pending) certificate for the apex domain, empty if none.
cert_arn() {
  site_us acm list-certificates --certificate-statuses "${1:-ISSUED}" \
    --query "CertificateSummaryList[?DomainName=='$DOMAIN'].CertificateArn | [0]" --output text \
    | sed 's/^None$//'
}

cmd="${1:-}"
case "$cmd" in
  cert)
    arn="$(cert_arn ISSUED)"
    if [[ -n "$arn" ]]; then
      echo "certificate already issued: $arn"
      exit 0
    fi
    arn="$(cert_arn PENDING_VALIDATION)"
    if [[ -z "$arn" ]]; then
      arn="$(site_us acm request-certificate --domain-name "$DOMAIN" \
        --subject-alternative-names "www.$DOMAIN" --validation-method DNS \
        --idempotency-token mestierewebsite \
        --tags Key=project,Value=mestiere \
        --query CertificateArn --output text)"
      echo "requested $arn"
    fi
    # ACM needs a few seconds before it exposes the validation records.
    records=""
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      records="$(site_us acm describe-certificate --certificate-arn "$arn" \
        --query 'Certificate.DomainValidationOptions[].ResourceRecord' --output json)"
      [[ "$records" != "null" && "$records" != *"null"* && "$records" != "[]" ]] && break
      sleep 3
    done
    changes="$(jq -c '{Comment: "mestiere.io ACM validation", Changes: (unique_by(.Name) | map(
      {Action: "UPSERT", ResourceRecordSet: {Name: .Name, Type: .Type, TTL: 300,
        ResourceRecords: [{Value: .Value}]}}))}' <<<"$records")"
    dns route53 change-resource-record-sets --hosted-zone-id "$DNS_ZONE_ID" \
      --change-batch "$changes" --query ChangeInfo.Status --output text
    echo "waiting for validation (usually a few minutes)..."
    site_us acm wait certificate-validated --certificate-arn "$arn"
    echo "certificate issued: $arn"
    ;;
  stack)
    arn="$(cert_arn ISSUED)"
    [[ -n "$arn" ]] || { echo "no issued certificate for $DOMAIN: run 'ops.sh cert' first" >&2; exit 1; }
    site cloudformation deploy \
      --stack-name "$STACK_NAME" \
      --template-file "$WEBSITE_DIR/infra/site.yaml" \
      --parameter-overrides "DomainName=$DOMAIN" "CertificateArn=$arn" \
      --tags project=mestiere \
      --capabilities CAPABILITY_NAMED_IAM \
      --no-fail-on-empty-changeset
    site cloudformation describe-stacks --stack-name "$STACK_NAME" \
      --query 'Stacks[0].Outputs' --output table
    ;;
  dns)
    target="$(stack_output DistributionDomainName)"
    changes="$(jq -n --arg apex "$DOMAIN." --arg target "$target." --arg zone "$CLOUDFRONT_ZONE_ID" '
      {Comment: "mestiere.io website", Changes: [($apex, "www." + $apex) as $name | ("A", "AAAA") as $t |
        {Action: "UPSERT", ResourceRecordSet: {Name: $name, Type: $t,
          AliasTarget: {HostedZoneId: $zone, DNSName: $target, EvaluateTargetHealth: false}}}]}')"
    dns route53 change-resource-record-sets --hosted-zone-id "$DNS_ZONE_ID" \
      --change-batch "$changes" --query ChangeInfo.Status --output text
    ;;
  deploy)
    (cd "$WEBSITE_DIR" && npm ci --no-audit --no-fund && npm run build)
    bucket="$(stack_output SiteBucketName)"
    # CloudFront keeps everything for a year (s-maxage) and the deploy
    # invalidates it; browsers revalidate pages and keep static files a while.
    PAGE='public, max-age=0, s-maxage=31536000, must-revalidate'
    STATIC='public, max-age=3600, s-maxage=31536000'
    # upload <cache-control> <content-type|-> <include patterns...>
    upload() {
      local cache="$1" type="$2"; shift 2
      local args=(--delete --exclude '*' --cache-control "$cache")
      [[ "$type" != "-" ]] && args+=(--content-type "$type")
      local p
      for p in "$@"; do args+=(--include "$p"); done
      site s3 sync "$DIST_DIR" "s3://$bucket" "${args[@]}" --only-show-errors
    }
    upload 'public, max-age=31536000, immutable' - 'fonts/*'
    upload "$STATIC" - 'assets/*' 'css/*' 'js/*' '*.png' '*.svg'
    upload "$PAGE" 'text/markdown; charset=utf-8' '*.md'
    upload "$PAGE" 'text/plain; charset=utf-8' '*.txt'
    upload "$PAGE" 'application/manifest+json' '*.webmanifest'
    upload "$PAGE" 'application/xml; charset=utf-8' '*.xml'
    upload "$PAGE" 'text/html; charset=utf-8' '*.html'
    site cloudfront create-invalidation --distribution-id "$(stack_output DistributionId)" \
      --paths '/*' --query Invalidation.Id --output text
    echo "deployed to $(stack_output SiteUrl)"
    ;;
  outputs)
    site cloudformation describe-stacks --stack-name "$STACK_NAME" \
      --query 'Stacks[0].Outputs' --output table
    ;;
  *)
    sed -n '2,19p' "$0"
    exit 1
    ;;
esac
