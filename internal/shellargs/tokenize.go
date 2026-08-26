package shellargs

import "fmt"

type quoteMode uint8

const (
	quoteNone quoteMode = iota
	quoteSingle
	quoteDouble
)

// Tokenize parses a shell-style argument string without executing shell code.
//
// Supported syntax:
//   - unquoted whitespace separates arguments
//   - single quotes preserve their contents literally
//   - double quotes preserve their contents literally
//   - backslash escapes the following character outside single quotes
//   - backslash escapes the following character inside double quotes
//   - empty quoted strings produce empty arguments
//
// It intentionally does not perform:
//   - variable expansion
//   - command substitution
//   - pathname expansion
//   - redirection
//   - pipelines
//   - command execution
func Tokenize(input string) ([]string, error) {
	var args []string
	var current []byte

	mode := quoteNone
	argumentStarted := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch mode {
		case quoteNone:
			switch ch {
			case '\'':
				mode = quoteSingle
				argumentStarted = true

			case '"':
				mode = quoteDouble
				argumentStarted = true

			case '\\':
				i++

				if i >= len(input) {
					return nil, fmt.Errorf(
						"trailing escape",
					)
				}

				current = append(
					current,
					input[i],
				)
				argumentStarted = true

			case ' ', '\t', '\n', '\r':
				if argumentStarted {
					args = append(
						args,
						string(current),
					)
					current = current[:0]
					argumentStarted = false
				}

			default:
				current = append(
					current,
					ch,
				)
				argumentStarted = true
			}

		case quoteSingle:
			if ch == '\'' {
				mode = quoteNone
				continue
			}

			current = append(
				current,
				ch,
			)

		case quoteDouble:
			switch ch {
			case '"':
				mode = quoteNone

			case '\\':
				i++

				if i >= len(input) {
					return nil, fmt.Errorf(
						"trailing escape in double quotes",
					)
				}

				current = append(
					current,
					input[i],
				)

			default:
				current = append(
					current,
					ch,
				)
			}
		}
	}

	switch mode {
	case quoteSingle:
		return nil, fmt.Errorf(
			"unterminated single quote",
		)

	case quoteDouble:
		return nil, fmt.Errorf(
			"unterminated double quote",
		)
	}

	if argumentStarted {
		args = append(
			args,
			string(current),
		)
	}

	return args, nil
}
