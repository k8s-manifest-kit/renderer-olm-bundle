#!/usr/bin/env bash

set -euo pipefail

# These advisories affect containerd's v1 CRI checkpoint import/restore paths.
# containerd v1 has no fixed release, and this renderer does not use those CRI
# paths. GO-2026-5932 is reported only at module level: the renderer uses the
# maintained ProtonMail OpenPGP fork instead of golang.org/x/crypto/openpgp,
# but a transitive dependency still requires the x/crypto module.
# Keep this list narrow: any new advisory must fail the check.
is_allowed() {
	case "$1" in
		GO-2026-5064|GO-2026-5338|GO-2026-5622|GO-2026-5932)
			return 0
			;;
		*)
			return 1
			;;
	esac
}

seen_ids=""
unexpected=0

if ! command -v jq >/dev/null 2>&1; then
	echo 'jq is required to parse govulncheck JSON output' >&2
	exit 1
fi

if ! ids=$(jq -r 'select(.finding != null and .finding.osv != null) | .finding.osv'); then
	echo 'failed to parse govulncheck JSON output' >&2
	exit 1
fi

while IFS= read -r id; do
	[[ -n "$id" ]] || continue

	case " $seen_ids " in
		*" $id "*)
			continue
			;;
	esac
	seen_ids+=" $id"

	if is_allowed "$id"; then
		printf 'Accepted known vulnerability: %s (https://pkg.go.dev/vuln/%s)\n' "$id" "$id"
	else
		printf 'Unexpected vulnerability: %s (https://pkg.go.dev/vuln/%s)\n' "$id" "$id" >&2
		unexpected=1
	fi
done <<< "$ids"

if ((unexpected)); then
	exit 1
fi
