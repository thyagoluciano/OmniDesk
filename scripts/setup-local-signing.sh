#!/usr/bin/env bash
# Creates (once) a local, self-signed code-signing certificate in the
# login keychain and uses it to sign OmniDesk.app going forward, instead
# of ad-hoc signing (`codesign -s -`).
#
# Why this matters: macOS ties Accessibility / Input Monitoring grants
# (needed by the Controle Remoto / KVM feature) to the app's code
# signature. Ad-hoc signing derives that identity from the binary's own
# content hash, which changes on every rebuild — so every rebuild
# silently invalidates any permission the user already granted, even
# though System Settings may still show a (stale, no-longer-matching)
# entry. Signing with a certificate ties the identity to the certificate
# instead, so rebuilds keep the same identity and permissions survive.
#
# Safe to re-run: skips certificate creation if one already exists.
set -e

CERT_NAME="OmniDesk Local Dev"
KEYCHAIN="${HOME}/Library/Keychains/login.keychain-db"

if security find-identity 2>/dev/null | grep -q "\"${CERT_NAME}\""; then
    echo "==> Certificado '${CERT_NAME}' já existe no keychain de login — nada a fazer."
    exit 0
fi

echo "==> Gerando certificado autoassinado local '${CERT_NAME}'..."
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

openssl req -x509 -newkey rsa:2048 \
    -keyout "${TMPDIR}/key.pem" -out "${TMPDIR}/cert.pem" \
    -days 3650 -nodes -subj "/CN=${CERT_NAME}" \
    -addext "keyUsage=critical,digitalSignature" \
    -addext "extendedKeyUsage=critical,codeSigning"

# -legacy: modern OpenSSL 3.x's default PKCS12 encryption isn't
# compatible with macOS's `security import` (fails with "MAC
# verification failed"); -legacy produces the older, compatible format.
openssl pkcs12 -export -legacy \
    -out "${TMPDIR}/cert.p12" \
    -inkey "${TMPDIR}/key.pem" -in "${TMPDIR}/cert.pem" \
    -passout pass:omnidesk

echo "==> Importando no keychain de login (autorizado só para o codesign)..."
security import "${TMPDIR}/cert.p12" -k "${KEYCHAIN}" -P omnidesk \
    -T /usr/bin/codesign -T /usr/bin/security

echo "==> Certificado '${CERT_NAME}' pronto. Builds futuros (scripts/build-macos-app.sh + omnidesk install) já vão usá-lo automaticamente."
