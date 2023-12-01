package cli

import (
	"fmt"
	"golang.org/x/term"
	"os"
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
	isTerm := term.IsTerminal(int(os.Stdout.Fd()))
	version = strings.Trim(version, " ")
	version = strings.Trim(version, "\n")

	var link string
	if isTerm {
		link = createHyperlink("http://localhost:" + port)
	} else {
		link = "http://localhost:" + port
	}

	buf := fmt.Sprintf(art, version, link)
	if isTerm {
		buf = blinkChars(buf, '.', '•', '·', '▪')
	}

	if ruleset == "" {
		buf += "\n [!] no ruleset specified.\n [!] for better performance, use a ruleset using --ruleset\n"
	}
	if isTerm {
		buf = colorizeNonASCII(buf)
	}
	return buf
}

func createHyperlink(url string) string {
	return fmt.Sprintf("\033[4m%s\033[0m", url)
}

func colorizeNonASCII(input string) string {
	result := ""
	for _, r := range input {
		if r > 127 {
			// If the character is non-ASCII, color it blue
			result += fmt.Sprintf("\033[34m%c\033[0m", r)
		} else {
			// ASCII characters remain unchanged
			result += string(r)
		}
	}
	return result
}

func blinkChars(input string, chars ...rune) string {
	result := ""
MAIN:
	for _, x := range input {
		for _, y := range chars {
			if x == y {
				result += fmt.Sprintf("\033[5m%s\033[0m", string(x))
				continue MAIN
			}
		}
		result += fmt.Sprintf("%s", string(x))
	}
	return result
}
