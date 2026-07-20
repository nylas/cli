#!/bin/sh
set -eu

repo_dir=${1:-.}
cd "$repo_dir"

found=0

excluded_dirs='(^|/)(\.git|vendor|node_modules|testdata|playwright-report|test-results|coverage|dist|build)/'

is_excluded_path() {
	printf '%s\n' "$1" | grep -Eq "$excluded_dirs"
}

is_text_candidate() {
	case "$1" in
		*_test.go | *.png | *.jpg | *.jpeg | *.gif | *.ico | *.pdf | *.zip | *.tar | *.gz | *.tgz | *.woff | *.woff2 | *.ttf | *.eot | *.mp4 | *.mov | *.sqlite | *.db)
			return 1
			;;
	esac

	return 0
}

list_scan_files() {
	git ls-files -co --exclude-standard 2>/dev/null | while IFS= read -r file; do
		[ -n "$file" ] || continue
		is_excluded_path "$file" && continue
		is_text_candidate "$file" || continue
		printf '%s\n' "$file"
	done
}

scan_content_locations() {
	pattern=$1

	list_scan_files | while IFS= read -r file; do
		grep -nEI "$pattern" "$file" 2>/dev/null | while IFS=: read -r line_no _; do
			[ -n "$line_no" ] || continue
			printf '%s:%s\n' "$file" "$line_no"
		done
	done | sort -u
}

is_config_candidate() {
	case "$1" in
		.env | .env.* | *.env | *.yaml | *.yml | *.json | *.toml | *.ini | *.conf | *.properties | *.tf | Dockerfile | docker-compose.yml | docker-compose.yaml)
			return 0
			;;
	esac

	return 1
}

scan_credential_locations() {
	credential_name="(api[_-]?key|apikey|password|passwd|secret|client[_-]?secret|access[_-]?token|refresh[_-]?token|private[_-]?key|bearer|aws[_-]?access[_-]?key[_-]?id)"
	source_pattern="$credential_name[[:space:]]*(:[[:space:]]+|=[[:space:]]*)['\"][A-Za-z0-9_./+=:@-]{8,}['\"]"
	config_pattern="$credential_name[[:space:]]*(:[[:space:]]+|=[[:space:]]*)['\"]?[A-Za-z0-9_./+=:@-]{8,}"

	list_scan_files | while IFS= read -r file; do
		pattern=$source_pattern
		if is_config_candidate "$file"; then
			pattern=$config_pattern
		fi

		grep -nEI "$pattern" "$file" 2>/dev/null | while IFS=: read -r line_no _; do
			[ -n "$line_no" ] || continue
			printf '%s:%s\n' "$file" "$line_no"
		done
	done | sort -u
}

filter_allowed_logging_locations() {
	while IFS= read -r location; do
		file=${location%:*}
		case "$file" in
			internal/cli/auth/token.go | internal/cli/doctor.go | internal/cli/ai/config_main.go)
				continue
				;;
		esac
		printf '%s\n' "$location"
	done
}

echo "=== Security Scan ==="
echo "Checking for hardcoded API keys..."
matches=$(scan_content_locations "nyk_v0[a-zA-Z0-9_]{20,}" || true)
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	echo "WARNING: Possible API key found!"
	found=1
else
	echo "OK: No API keys found"
fi

echo ""
echo "Checking for credential patterns..."
matches=$(scan_credential_locations || true)
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	echo "WARNING: Possible credentials found!"
	found=1
else
	echo "OK: No hardcoded credentials"
fi

echo ""
echo "Checking for full credential logging..."
logging_pattern="fmt\.(Print|Fprint|Sprint)[A-Za-z]*\([^)]*([A-Za-z0-9_]*[Aa]pi[Kk]ey|api_key)"
matches=$(scan_content_locations "$logging_pattern" | filter_allowed_logging_locations || true)
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	echo "WARNING: Possible credential logging!"
	found=1
else
	echo "OK: No credential logging"
fi

echo ""
echo "Checking staged files..."
# --diff-filter=d excludes deletions: removing a file cannot leak its contents,
# and staged deletions of e.g. test fixtures would otherwise trip this check.
matches=$(git diff --cached --name-only --diff-filter=d 2>/dev/null | grep -E '(^|/)\.env([^/]*|$)|\.(key|pem|json)$|(^|/)secrets/' || true)
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	echo "WARNING: Sensitive file staged!"
	found=1
else
	echo "OK: No sensitive files staged"
fi

echo ""
echo "Checking tracked files..."
matches=$(git ls-files 2>/dev/null | grep -Ei '(^|/)\.env([^/]*|$)|\.(key|pem)$|(^|/)secrets/|(^|/)[^/]*(secret|credential|credentials|service-account|private-key|private)[^/]*\.json$' || true)
if [ -n "$matches" ]; then
	printf '%s\n' "$matches"
	echo "WARNING: Sensitive tracked files found!"
	found=1
else
	echo "OK: No sensitive tracked files"
fi

echo ""
if [ "$found" -ne 0 ]; then
	echo "Security scan failed"
	exit 1
fi

echo "=== Security scan complete ==="
