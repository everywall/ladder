package cli

import (
	"fmt"
	"strings"
)

var art string = `
 _____╬═╬____________________________________________
 |_|__╬═╬___|___|___|___| EVERYWALL |___|___|___|___|
 |___|╬═╬|___▄▄▌   ▄▄▄· ·▄▄▄▄  ·▄▄▄▄  ▄▄▄ .▄▄▄  __|_|
 |_|__╬═╬___|██•  ▐█ ▀█ ██▪ ██ ██▪ ██ ▀▄.▀·▀▄ █·|___|
 |___|╬═╬|___██▪  ▄█▀▀█ ▐█· ▐█▌▐█· ▐█▌▐▀▀▪▄▐▀▀▄ __|_|
 |_|__╬═╬___|▐█▌▐▌▐█ ▪▐▌██. ██ ██. ██ ▐█▄▄▌▐█•█▌|___|
 |___|╬═╬|___.▀▀▀  ▀  ▀ ▀▀▀▀▀• ▀▀▀▀▀•  ▀▀▀ .▀  ▀__|_|
 |_|__╬═╬___|___|___|_ VERSION %-7s__|___|___|___|
 |___|╬═╬|____|___|___|___|___|___|___|___|___|___|_|
      ╬═╬
      ╬═╬          %s
 `

func StartupMessage(version string, port string, ruleset string) string {
	version = strings.Trim(version, " ")
	version = strings.Trim(version, "\n")
	link := createHyperlink("http://localhost:" + port)
	buf := fmt.Sprintf(art, version, link)
	if ruleset == "" {
		buf += "\n ! no ruleset specified.\n > for better performance, use a ruleset using --ruleset\n"
	} else {
		buf += fmt.Sprintf("\n > using ruleset: %s\n", ruleset)
	}
	return colorizeNonASCII(buf)
}

func createHyperlink(url string) string {
	//return fmt.Sprintf("\033]8;;%s\a%s\033]8;;\a", url, url)
	return fmt.Sprintf("\033[4m%s\033[0m", url)
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
