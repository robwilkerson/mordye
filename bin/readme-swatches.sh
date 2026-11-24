#!/usr/bin/env bash
# Draws the README's terminal color comparisons as SVG swatches in assets/.
#
#   bin/readme-swatches.sh
#
# Mordye's row comes from running Mordye on the Dux fixture, so rerun this
# after any change to how Mordye picks terminal colors. The official row is
# read from the bundled palette file, so it can't drift from what Mordye
# ships. Dux's rows are fixed: the palette Dux generated for Catppuccin, read
# from Dux's cache (~/.cache/dms/dms-colors.json, the dark color0-7 of its
# dank16 palette) on 2026-11-15.
set -euo pipefail
cd "$(dirname "$0")/.."

slots=(black red green yellow blue magenta cyan white)

# mordye_row FLAVOR ACCENT THEME: Mordye's eight normal colors for a Dux
# fixture, with the flavor and accent chosen as Dux settings would.
mordye_row() {
	local settings
	settings=$(mktemp)
	printf '{"currentThemeName":"custom","registryThemeVariants":{"%s":{"dark":{"flavor":"%s","accent":"%s"}}}}' \
		"$3" "$1" "$2" >"$settings"
	go run ./cmd/mordye dux --theme "adapter/dux/testdata/$3.json" --settings "$settings" |
		jq -r '.terminal.normal | [.black, .red, .green, .yellow, .blue, .magenta, .cyan, .white] | join(" ")'
	rm -f "$settings"
}

# palette_row ID: a bundled palette's eight normal colors, lowercased as
# Mordye writes them.
palette_row() {
	jq -r '.terminal.normal | [.black, .red, .green, .yellow, .blue, .magenta, .cyan, .white] | map(ascii_downcase) | join(" ")' \
		"palettes/$1.json"
}

# svg FILE TITLE BACKGROUND TEXT DIM ROWS...: each row is
# "label|sublabel|eight hex colors".
svg() {
	local file=$1 title=$2 bg=$3 text=$4 dim=$5
	shift 5
	local width=870 height=$((44 + 84 * $# + 4))
	{
		printf '<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="title">\n' \
			$width $height $width $height
		printf '  <title id="title">%s</title>\n' "$title"
		printf '  <rect width="%d" height="%d" rx="12" fill="%s"/>\n' $width $height "$bg"
		printf '  <g font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif" fill="%s" font-size="13" text-anchor="middle">\n' "$dim"
		for i in "${!slots[@]}"; do
			printf '    <text x="%d" y="28">%s</text>\n' $((246 + 80 * i)) "${slots[i]}"
		done
		printf '  </g>\n'
		local r=0 top label sub colors hex
		for row in "$@"; do
			IFS='|' read -r label sub colors <<<"$row"
			top=$((44 + 84 * r))
			printf '  <text x="20" y="%d" fill="%s" font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif" font-size="15" font-weight="600">%s</text>\n' \
				$((top + 26)) "$text" "$label"
			printf '  <text x="20" y="%d" fill="%s" font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif" font-size="12">%s</text>\n' \
				$((top + 46)) "$dim" "$sub"
			local i=0
			for hex in $colors; do
				printf '  <rect x="%d" y="%d" width="72" height="44" rx="6" fill="%s" stroke="%s" stroke-opacity="0.5"/>\n' \
					$((210 + 80 * i)) "$top" "$hex" "$dim"
				printf '  <text x="%d" y="%d" fill="%s" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="11" text-anchor="middle">%s</text>\n' \
					$((246 + 80 * i)) $((top + 62)) "$dim" "$hex"
				i=$((i + 1))
			done
			r=$((r + 1))
		done
		printf '</svg>\n'
	} >"$file"
	echo "wrote $file"
}

# Catppuccin's official colors are its bundled palette, from its kitty port.
svg assets/catppuccin-mocha-terminal.svg \
	"Catppuccin Mocha's eight terminal colors: the official palette, Mordye's under either accent, and Dux's under the mauve and green accents." \
	"#1e1e2e" "#cdd6f4" "#a6adc8" \
	"Catppuccin|official palette|$(palette_row catppuccin-mocha)" \
	"Mordye|mauve or green accent|$(mordye_row mocha mauve catppuccin)" \
	"Dux|mauve accent|#181825 #ff728c #7bf797 #ffdb72 #b88eea #55307f #cba6f7 #f6efff" \
	"Dux|green accent|#181825 #ff9372 #74e371 #ffee72 #92d78c #2f5f36 #a6e3a1 #e0eee2"
