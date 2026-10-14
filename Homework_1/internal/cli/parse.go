package cli

import (
	"fmt"
	"strings"
	"unicode"
)

func splitCommand(line string) []string {
	var args []string
	var buf strings.Builder
	inQuote := false

	for _, r := range strings.TrimSpace(line) {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if buf.Len() > 0 {
				args = append(args, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		args = append(args, buf.String())
	}
	return args
}

type flags struct {
	values map[string]string
	bools  map[string]bool
}

func parseFlags(args []string) (flags, error) {
	f := flags{
		values: make(map[string]string),
		bools:  make(map[string]bool),
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return flags{}, fmt.Errorf("неожиданный аргумент %q", arg)
		}
		name := strings.TrimPrefix(arg, "--")
		if name == "" {
			return flags{}, fmt.Errorf("пустой флаг")
		}
		if name == "overdue" {
			f.bools[name] = true
			continue
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return flags{}, fmt.Errorf("флаг --%s требует значение", name)
		}
		i++
		f.values[name] = args[i]
	}
	return f, nil
}
