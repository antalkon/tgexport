package cli

const usageText = `tgexport — convert Telegram Desktop JSON chat exports to text, HTML and more.

Usage:
  tgexport [flags] <result.json | export-folder>
  tgexport stats <result.json | export-folder>
  tgexport version

By default the chat is written as plain text to <chat name>.txt in the
current directory, one message per line:

  2026-10-22 22:09:09 Alice: hello there

Output:
  -f, --format string     txt | html | md | csv | json (default "txt")
      --html              shortcut for --format html
  -o, --output path       output file, directory (ending in /), or "-" for stdout
                          (default: current directory)
  -q, --quiet             do not print a summary line per exported chat

Filters:
      --from date         keep messages on or after date (YYYY-MM-DD or "YYYY-MM-DD HH:MM")
      --to date           keep messages on or before date (inclusive)
  -u, --user name         keep messages by this author name or ID (repeatable, comma-separated)
      --no-service        drop service messages (joins, pins, calls)
      --no-media          drop media placeholders like [photo]; skip media-only messages
      --chat name|id      pick one chat from a full account export

Formatting:
      --layout string     txt line template (default "{date} {time} {user}: {text}")
                          placeholders: {date} {time} {user} {user_id} {id} {text}
      --date-format str   Go time layout for {date} (default "2006-01-02")
      --time-format str   Go time layout for {time} (default "15:04:05")
      --me name           in HTML, draw this author's messages on the right

Other:
  -v, --version           print version and exit
  -h, --help              show this help

Examples:
  tgexport result.json
  tgexport ~/Downloads/ChatExport_2026-10-31 --html
  tgexport result.json -f md -o exports/
  tgexport result.json --from 2026-10-01 --to 2026-10-15 -u Alice
  tgexport result.json --layout "{date}:{time}:{user} {text}"
  tgexport result.json -o - | grep -i deadline

Docs: https://github.com/antalkon/tgexport
`
