#!/usr/bin/env bash
# Validates charts/ostendorf.tech: a must-succeed set for its default values and every
# documented non-default combination, and a must-fail set proving every
# templates/deployment.yaml (the API's), templates/deployment-frontend.yaml,
# templates/deployment-feed.yaml, templates/httproute.yaml and the three NetworkPolicy
# templates' fail/required guard actually rejects the values it is meant to reject, plus the
# values.schema.json refusals of the feed's OpenBao auth values (a plain token value
# above all), of its postCount and postsPerRequest ranges and of the goMemLimit form,
# and the guard that keeps each goMemLimit below its memory limit. Shared by
# task lint:helm and task validate:kubeconform so CI and a local run check the exact same
# succeed combinations. Every combination, succeed and fail alike, also carries the
# baseline below: existingSecret.name has no usable default and is enforced by a
# required() guard, so even "default values" fails to render without it.
#
# feed.config.openBaoAddr and feed.config.openBaoRole have no usable default either, but
# their own required() guards only fire once feed.enabled is true (feed.enabled itself
# defaults to false), so they are added on top of the baseline only for the "feed enabled"
# combination and the two must-fail cases that exercise them, not for every combination.
#
# Takes exactly one mode argument:
#   lint        - helm lint over the must-succeed set, then helm template over the
#                 must-fail set, one job per check (Taskfile's lint:helm / CI's helm-lint
#                 job)
#   kubeconform - helm template piped into kubeconform, its own job (CI's kubeconform job)
# kept as two modes of one script, not two near-duplicate scripts, so the combinations
# below live in exactly one place.
#
# The must-fail set is checked with helm template, never helm lint: a template's fail or
# required() call is only logged by helm lint, which still reports "0 chart(s) failed" and
# exits zero, so a lint pass proves nothing about a guard built on either. helm template
# propagates that failure as a non-zero exit, so each must-fail entry asserts both the
# non-zero exit and the exact message its guard names, which also catches a guard that
# starts failing for the wrong reason, or a different guard firing instead of the one
# this case means to exercise.
#
# kubeconform's own default schema location has no schema for a Gateway API HTTPRoute: it
# is a CustomResourceDefinition, not a built in Kubernetes type, so the default registry
# never carries it. The "-schema-location default" below keeps every built in resource on
# kubeconform's normal, self updating registry, and the second "-schema-location" adds
# Datree's CRDs-catalog (https://github.com/datreeio/CRDs-catalog) as a second lookup for
# anything the first one does not resolve, HTTPRoute included.
set -euo pipefail

mode="${1:?usage: validate-chart.sh <lint|kubeconform>}"

case "$mode" in
  lint | kubeconform) ;;
  *)
    echo "validate-chart: unknown mode '${mode}', expected 'lint' or 'kubeconform'" >&2
    exit 1
    ;;
esac

chart="charts/ostendorf.tech"

# existingSecret.name has no usable default (see values.yaml) and is enforced by
# templates/_helpers.tpl's required() call, so every combination below needs it set, the
# default values combination included.
baseline="existingSecret.name=ostendorf-tech-secrets"

# The two external routing combinations, each valid on its own: ingress enabled with its
# own required trustedProxies guard satisfied, httpRoute enabled with both of its own
# required parentRefs fields set.
ingress_valid="config.trustedProxies=10.0.0.0/8,ingress.enabled=true,ingress.className=nginx"
ingress_valid="${ingress_valid},ingress.hosts[0].host=ostendorf.tech"
httproute_valid="httpRoute.enabled=true,httpRoute.hostnames[0]=ostendorf.tech"
httproute_valid="${httproute_valid},httpRoute.parentRefs.name=shared-gateway,httpRoute.parentRefs.namespace=gateway-system"

# feed.enabled defaults to false; feed.config.openBaoAddr and feed.config.openBaoRole have
# no usable default and are enforced by templates/_helpers.tpl's required() calls, but only
# once feed.enabled is true, so this combination sets all three together.
feed_valid="feed.enabled=true,feed.config.openBaoAddr=https://openbao.openbao.svc.cluster.local:8200,feed.config.openBaoRole=ostendorf-tech-feed"

# The feed service with the token auth method: no role, the token from an existing
# Secret reference only, never a value.
feed_base="feed.enabled=true,feed.config.openBaoAddr=https://openbao.openbao.svc.cluster.local:8200"
feed_token_valid="${feed_base},feed.config.openBaoAuthMethod=token"
feed_token_valid="${feed_token_valid},feed.openBaoToken.existingSecret=feed-openbao-token,feed.openBaoToken.existingSecretKey=token"

# The feed service with a custom secret path and custom key names inside it.
feed_secret_valid="${feed_valid},feed.config.openBaoSecretPath=kv/data/instagram"
feed_secret_valid="${feed_secret_valid},feed.config.openBaoSecretKeyAccessToken=ig_token,feed.config.openBaoSecretKeyRefreshedAt=ig_refreshed"

# config.smtpHost defaults to empty, so the default values combination renders without
# SMTP settings (the contact form answers 503); this one sets the server and the secret
# keys the contact form needs.
contact_valid="config.smtpHost=mail.ostendorf.cloud,config.smtpPort=587"
contact_valid="${contact_valid},existingSecret.challengeKeyKey=CHALLENGE_KEY"
contact_valid="${contact_valid},existingSecret.smtpUsernameKey=SMTP_USERNAME,existingSecret.smtpPasswordKey=SMTP_PASSWORD"

# The NetworkPolicies of the API and the frontend: both default to disabled because each
# needs the ingress controller's namespace, which has no usable default. This combination
# enables both with that selector, an optional metrics selector, an SMTP CIDR for the
# API, Umami with its egress target for the frontend, and the feed so that the API's egress
# to the feed pods renders too.
np_valid="networkPolicy.enabled=true,networkPolicy.ingressController.namespaceSelector.kubernetes\.io/metadata\.name=ingress-nginx"
np_valid="${np_valid},networkPolicy.ingressController.podSelector.app\.kubernetes\.io/name=ingress-nginx"
np_valid="${np_valid},networkPolicy.metrics.namespaceSelector.kubernetes\.io/metadata\.name=monitoring"
np_valid="${np_valid},networkPolicy.smtp.cidr=192.0.2.10/32,config.smtpHost=mail.ostendorf.cloud"
np_valid="${np_valid},frontend.networkPolicy.enabled=true"
np_valid="${np_valid},frontend.networkPolicy.ingressController.namespaceSelector.kubernetes\.io/metadata\.name=ingress-nginx"
np_valid="${np_valid},frontend.networkPolicy.metrics.podSelector.app\.kubernetes\.io/name=prometheus"
np_valid="${np_valid},config.umamiUrl=http://umami.umami.svc.cluster.local:3000,config.umamiWebsiteId=0f1e2d3c-4b5a-4968-8776-655443322110"
np_valid="${np_valid},frontend.networkPolicy.umami.namespaceSelector.kubernetes\.io/metadata\.name=umami,frontend.networkPolicy.umami.port=3000"
np_valid="${np_valid},${feed_valid}"

# The frontend with every optional value set: more than one replica, Umami with its
# optional timeout and queue size, a custom image tag.
frontend_valid="frontend.replicaCount=3,frontend.image.tag=v0.1.0,config.umamiUrl=https://umami.ostendorf.cloud"
frontend_valid="${frontend_valid},config.umamiWebsiteId=0f1e2d3c-4b5a-4968-8776-655443322110"
frontend_valid="${frontend_valid},config.umamiTimeout=2s,config.umamiQueueSize=100,frontend.goMemLimit=80MiB"

names=(
  "default values"
  "ingress enabled"
  "httpRoute enabled"
  "feed enabled"
  "feed enabled with the token auth method"
  "feed enabled with a custom secret path and key names"
  "contact form configured"
  "frontend with every optional value set"
  "API and frontend NetworkPolicies enabled"
  "API and frontend NetworkPolicies enabled behind an ingress"
  "API and frontend NetworkPolicies enabled behind an httpRoute"
)

# Each entry is one combination's --set arguments, matching the corresponding
# charts/ostendorf.tech/tests/*_test.yaml suite's own "set:" block, always layered on top
# of baseline above.
sets=(
  "${baseline}"
  "${baseline},${ingress_valid}"
  "${baseline},${httproute_valid}"
  "${baseline},${feed_valid}"
  "${baseline},${feed_token_valid}"
  "${baseline},${feed_secret_valid}"
  "${baseline},${contact_valid}"
  "${baseline},${frontend_valid}"
  "${baseline},${np_valid}"
  "${baseline},${np_valid},${ingress_valid}"
  "${baseline},${np_valid},${httproute_valid}"
)

# kubeconform's default schema location carries no Gateway API schemas. This second
# location is only consulted for a kind the first one misses, HTTPRoute included; see
# this file's own header comment.
gateway_api_schema_location='https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json'

for i in "${!names[@]}"; do
  name="${names[$i]}"
  set_args="${sets[$i]}"

  echo "==> validating combination (${mode}): ${name}"

  case "$mode" in
    lint)
      if ! helm lint "${chart}" --set "${set_args}"; then
        echo "validate-chart: helm lint failed for combination: ${name}" >&2
        exit 1
      fi
      ;;
    kubeconform)
      if ! helm template "${chart}" --set "${set_args}" |
        kubeconform -strict -summary \
          -schema-location default \
          -schema-location "${gateway_api_schema_location}"; then
        echo "validate-chart: kubeconform failed for combination: ${name}" >&2
        exit 1
      fi
      ;;
  esac
done

if [ "$mode" = "lint" ]; then
  # Each entry names the guard it exercises in templates/deployment.yaml,
  # templates/deployment-feed.yaml or templates/httproute.yaml, its --set arguments on
  # top of baseline above (overriding exactly one baseline value where the guard needs
  # that value missing instead of present), and the exact message the guard's fail or
  # required() call names. Every other guard's own required values stay satisfied, so
  # each case fails for its own guard alone, not a different one firing first.
  fail_names=(
    "both ingress.enabled and httpRoute.enabled true"
    "replicaCount above 1"
    "ingress.enabled true with config.trustedProxies left empty"
    "httpRoute.enabled true with parentRefs.name left empty"
    "httpRoute.enabled true with parentRefs.namespace left empty"
    "existingSecret.name left empty"
    "feed.enabled true with feed.config.openBaoAddr left empty"
    "feed.enabled true with feed.config.openBaoRole left empty"
    "feed token auth method without a token Secret reference"
    "feed token auth method with an empty token Secret key"
    "feed token auth method with a Kubernetes role"
    "feed kubernetes auth method with a token Secret reference"
    "feed unknown auth method"
    "feed token given as a plain value"
    "feed removed tokenPath value"
    "feed secret key name with a slash"
    "feed postsPerRequest above Meta's maximum"
    "feed postsPerRequest of zero"
    "feed negative postCount"
    "feed postCount above the maximum"
    "goMemLimit at the website's memory limit"
    "feed.goMemLimit above the feed's memory limit"
    "feed.goMemLimit not in Go's form"
    "config.allowPlainHttp true with an https config.siteUrl"
    "config.allowPlainHttp true with an upper case HTTPS config.siteUrl"
    "frontend.goMemLimit at the frontend's memory limit"
    "frontend.goMemLimit not in Go's form"
    "frontend.networkPolicy.umami.port out of range"
    "config.umamiQueueSize negative"
    "networkPolicy.enabled true without an ingress controller namespaceSelector"
    "frontend.networkPolicy.enabled true without an ingress controller namespaceSelector"
    "networkPolicy.enabled true with config.smtpHost set and no networkPolicy.smtp.cidr"
    "frontend.networkPolicy.enabled true with config.umamiUrl set and no Umami egress target"
  )

  fail_sets=(
    "${baseline},${ingress_valid},${httproute_valid}"
    "${baseline},replicaCount=2"
    "${baseline},ingress.enabled=true,ingress.className=nginx,ingress.hosts[0].host=ostendorf.tech,ingress.hosts[0].paths[0].path=/,ingress.hosts[0].paths[0].pathType=Prefix"
    "${baseline},httpRoute.enabled=true,httpRoute.parentRefs.namespace=gateway-system"
    "${baseline},httpRoute.enabled=true,httpRoute.parentRefs.name=shared-gateway"
    "replicaCount=1"
    "${baseline},feed.enabled=true,feed.config.openBaoRole=ostendorf-tech-feed"
    "${baseline},feed.enabled=true,feed.config.openBaoAddr=https://openbao.openbao.svc.cluster.local:8200"
    "${baseline},${feed_base},feed.config.openBaoAuthMethod=token"
    "${baseline},${feed_token_valid},feed.openBaoToken.existingSecretKey="
    "${baseline},${feed_token_valid},feed.config.openBaoRole=ostendorf-tech-feed"
    "${baseline},${feed_valid},feed.openBaoToken.existingSecret=feed-openbao-token"
    "${baseline},${feed_valid},feed.config.openBaoAuthMethod=approle"
    "${baseline},${feed_token_valid},feed.openBaoToken.value=not-a-real-token"
    "${baseline},${feed_valid},feed.config.tokenPath=old/data/path"
    "${baseline},${feed_valid},feed.config.openBaoSecretKeyAccessToken=ig/token"
    "${baseline},${feed_valid},feed.config.postsPerRequest=101"
    "${baseline},${feed_valid},feed.config.postsPerRequest=0"
    "${baseline},${feed_valid},feed.config.postCount=-1"
    "${baseline},${feed_valid},feed.config.postCount=10001"
    "${baseline},goMemLimit=192MiB"
    "${baseline},${feed_valid},feed.goMemLimit=200MiB"
    "${baseline},${feed_valid},feed.goMemLimit=144M"
    "${baseline},config.allowPlainHttp=true,config.siteUrl=https://ostendorf.tech"
    "${baseline},config.allowPlainHttp=true,config.siteUrl=HTTPS://ostendorf.tech"
    "${baseline},frontend.goMemLimit=96MiB"
    "${baseline},frontend.goMemLimit=86M"
    "${baseline},frontend.networkPolicy.umami.port=0"
    "${baseline},config.umamiQueueSize=-1"
    "${baseline},networkPolicy.enabled=true"
    "${baseline},frontend.networkPolicy.enabled=true"
    "${baseline},${np_valid},networkPolicy.smtp.cidr="
    "${baseline},${np_valid},frontend.networkPolicy.umami.namespaceSelector=null"
  )

  fail_messages=(
    "ingress.enabled and httpRoute.enabled cannot both be true: pick one external route and turn the other off."
    "replicaCount must stay at 1: govue's contact form keeps its rate limits, its global mail ceiling and its proof of work replay set in the memory of one process. Every replica has its own, so two replicas double every budget, the mail ceiling included, and a solved challenge can be used once per replica. Raise replicaCount only after those move to a shared store; see AGENTS.md."
    "config.trustedProxies must be set when ingress.enabled is true: clientip.NewResolver with no trusted prefixes attributes every request to its TCP peer, which behind this chart's own Ingress is the ingress controller's pod IP. Every rate limiter then keys on that one address, collapsing the whole site's traffic into one shared bucket (roughly 20 page loads per minute site-wide, and five contact submissions blocking the form for every visitor for ContactRateLimitPer). Set it to the ingress controller's pod or Service CIDR; see README.md."
    "httpRoute.parentRefs.name is required when httpRoute.enabled is true"
    "httpRoute.parentRefs.namespace is required when httpRoute.enabled is true: the Gateway this HTTPRoute attaches to usually lives in a different namespace than this release"
    "existingSecret.name is required: set it to the name of an existing Secret carrying CHALLENGE_KEY and the SMTP credentials, without which the contact form answers 503."
    "feed.config.openBaoAddr is required: set it to the OpenBao address the feed service authenticates against. There is no default, since none could ever point at an OpenBao installation that actually exists."
    "feed.config.openBaoRole is required: set it to the OpenBao Kubernetes auth role the feed service logs in as. There is no default, since none could ever name a role that actually exists in your OpenBao installation."
    "feed.openBaoToken.existingSecret is required with feed.config.openBaoAuthMethod=token: set it to the name of an existing Secret holding the OpenBao token."
    "feed.openBaoToken.existingSecretKey is required with feed.config.openBaoAuthMethod=token: set it to the key of the OpenBao token inside that Secret."
    "feed.config.openBaoRole must not be set with feed.config.openBaoAuthMethod=token: the role belongs to the kubernetes method."
    "feed.openBaoToken.existingSecret must not be set with feed.config.openBaoAuthMethod=kubernetes: the token belongs to the token method."
    "at '/feed/config/openBaoAuthMethod': value must be one of 'kubernetes', 'token'"
    "at '/feed/openBaoToken': additional properties 'value' not allowed"
    "at '/feed/config/tokenPath': false schema"
    "at '/feed/config/openBaoSecretKeyAccessToken'"
    "at '/feed/config/postsPerRequest': maximum"
    "at '/feed/config/postsPerRequest': minimum"
    "at '/feed/config/postCount': minimum"
    "at '/feed/config/postCount': maximum"
    "goMemLimit must be below resources.limits.memory"
    "feed.goMemLimit must be below feed.resources.limits.memory"
    "at '/feed/goMemLimit'"
    "config.allowPlainHttp must not be true when config.siteUrl is https: it turns HSTS off, which a site served over https must keep. Set it only for a plain http development run, or serve the site over plain http."
    "config.allowPlainHttp must not be true when config.siteUrl is https: it turns HSTS off, which a site served over https must keep. Set it only for a plain http development run, or serve the site over plain http."
    "frontend.goMemLimit must be below frontend.resources.limits.memory"
    "at '/frontend/goMemLimit'"
    "at '/frontend/networkPolicy/umami/port': minimum"
    "at '/config/umamiQueueSize': minimum"
    "networkPolicy.ingressController.namespaceSelector is required when networkPolicy.enabled is true"
    "frontend.networkPolicy.ingressController.namespaceSelector is required when frontend.networkPolicy.enabled is true"
    "networkPolicy.smtp.cidr is required when networkPolicy.enabled and config.smtpHost are set"
    "frontend.networkPolicy.umami needs a cidr, a namespaceSelector or a podSelector when frontend.networkPolicy.enabled and config.umamiUrl are set"
  )

  for i in "${!fail_names[@]}"; do
    name="${fail_names[$i]}"
    set_args="${fail_sets[$i]}"
    expected="${fail_messages[$i]}"

    echo "==> validating must-fail combination (template): ${name}"

    render_output=""
    render_status=0
    render_output="$(helm template "${chart}" --set "${set_args}" 2>&1)" || render_status=$?

    if [ "${render_status}" -eq 0 ]; then
      echo "validate-chart: combination '${name}' was expected to fail its guard but rendered successfully" >&2
      exit 1
    fi

    if [[ "${render_output}" != *"${expected}"* ]]; then
      echo "validate-chart: combination '${name}' failed, but not with its guard's own message" >&2
      echo "validate-chart: expected the output to contain: ${expected}" >&2
      echo "validate-chart: actual output:" >&2
      echo "${render_output}" >&2
      exit 1
    fi
  done
fi

echo "all combinations valid (${mode})"
