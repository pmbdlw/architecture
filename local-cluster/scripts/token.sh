#!/usr/bin/env bash
# 用“授权码 + PKCE”流程从认证中心获取访问令牌（与前端 SPA 的登录流程完全相同，只是用 curl 代替浏览器）。
#   scripts/token.sh go alice            → 输出 access_token
#   TOKEN=$(scripts/token.sh spring bob)
# 演示账号：admin(uid=1) / alice(uid=1001) / bob(uid=1002)，密码均为 password
source "$(dirname "$0")/lib.sh"

STACK=$(stack_arg "${1:-}")
USERNAME=${2:-alice}
PASSWORD=${3:-password}
CLIENT_ID=acme-web
REDIRECT_URI=http://localhost:3000/callback
SCOPE="openid order.read order.write"
need curl openssl

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }
VERIFIER=$(openssl rand 48 | b64url)
CHALLENGE=$(printf '%s' "$VERIFIER" | openssl dgst -binary -sha256 | b64url)
STATE=$(openssl rand -hex 8)

JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT
C=(curl -s --noproxy '*' -b "$JAR" -c "$JAR" -o /dev/null -w '%{redirect_url}')
PARAMS=(
  --data-urlencode response_type=code --data-urlencode "client_id=$CLIENT_ID"
  --data-urlencode "redirect_uri=$REDIRECT_URI" --data-urlencode "scope=$SCOPE"
  --data-urlencode "state=$STATE" --data-urlencode "code_challenge=$CHALLENGE"
  --data-urlencode code_challenge_method=S256
)

code_from() { sed -n 's/.*[?&]code=\([^&]*\).*/\1/p' <<<"$1"; }

if [[ $STACK == go ]]; then
  # Go 认证中心：登录表单直接提交到授权端点，成功后 302 回 redirect_uri?code=…
  loc=$("${C[@]}" -X POST "$AUTH_URL/oauth2/authorize" "${PARAMS[@]}" \
    --data-urlencode "username=$USERNAME" --data-urlencode "password=$PASSWORD")
else
  # Spring Authorization Server：授权端点 → 登录页（带 CSRF）→ 登录 → 回到授权端点 → 302 回 redirect_uri?code=…
  "${C[@]}" -G "$AUTH_URL/oauth2/authorize" "${PARAMS[@]}" >/dev/null
  csrf=$(curl -s --noproxy '*' -b "$JAR" -c "$JAR" "$AUTH_URL/login" | sed -n 's/.*name="_csrf"[^>]*value="\([^"]*\)".*/\1/p' | head -n1)
  [[ -n "$csrf" ]] || die "未取到登录页的 CSRF 令牌（认证中心是否就绪？$AUTH_URL/login）"
  loc=$("${C[@]}" -X POST "$AUTH_URL/login" --data-urlencode "username=$USERNAME" \
    --data-urlencode "password=$PASSWORD" --data-urlencode "_csrf=$csrf")
  [[ "$loc" == *"/oauth2/authorize"* ]] || die "登录失败（用户名或密码错误？）"
  loc=$("${C[@]}" "$loc")
fi

CODE=$(code_from "$loc")
[[ -n "$CODE" ]] || die "未获得授权码：$loc"

resp=$(curl -s --noproxy '*' -X POST "$AUTH_URL/oauth2/token" \
  --data-urlencode grant_type=authorization_code --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "code=$CODE" --data-urlencode "redirect_uri=$REDIRECT_URI" \
  --data-urlencode "code_verifier=$VERIFIER")
TOKEN=$(sed -n 's/.*"access_token" *: *"\([^"]*\)".*/\1/p' <<<"$resp")
[[ -n "$TOKEN" ]] || die "换取令牌失败：$resp"
echo "$TOKEN"
