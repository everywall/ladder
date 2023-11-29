package cli

import "fmt"

var art string = `
 _____╬═╬____________________________________________
 |_|__╬═╬___|___|___|___| EVERYWALL |___|___|___|___|
 |___|╬═╬|___▄▄▌   ▄▄▄· ·▄▄▄▄  ·▄▄▄▄  ▄▄▄ .▄▄▄  __|_|
 |_|__╬═╬___|██•  ▐█ ▀█ ██▪ ██ ██▪ ██ ▀▄.▀·▀▄ █·|___|
 |___|╬═╬|___██▪  ▄█▀▀█ ▐█· ▐█▌▐█· ▐█▌▐▀▀▪▄▐▀▀▄ __|_|
 |_|__╬═╬___|▐█▌▐▌▐█ ▪▐▌██. ██ ██. ██ ▐█▄▄▌▐█•█▌|___|
 |___|╬═╬|___.▀▀▀  ▀  ▀ ▀▀▀▀▀• ▀▀▀▀▀•  ▀▀▀ .▀  ▀__|_|
 |_|__╬═╬___|___|___|__ VERSION %s __|___|___|___|
 |___|╬═╬|____|___|___|___|___|___|___|___|___|___|_|
 `

func StartupMessage(version string, port string, ruleset string) string {
	buf := fmt.Sprintf(art, version)
	buf += fmt.Sprintf("\n > listening on http://localhost:%s\n", port)
	if ruleset == "" {
		buf += " ! no ruleset specified.\n > for better performance, use a ruleset using --ruleset\n"
	} else {
		buf += fmt.Sprintf(" > using ruleset: %s\n", ruleset)
	}
	return colorizeNonASCII(buf)
}

func colorizeNonASCII(input string) string {
	result := ""
	for _, r := range input {
		if r > 127 {
			// If the character is non-ASCII, color it blue
			result += fmt.Sprintf("\033[94m%c\033[0m", r)
		} else {
			// ASCII characters remain unchanged
			result += string(r)
		}
	}
	return result
}
