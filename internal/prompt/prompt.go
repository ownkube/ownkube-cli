package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ReadSecret prompts for sensitive input (e.g., API key) with hidden echo.
// Falls back to plain text input if terminal masking is unavailable.
func ReadSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return readLine(os.Stdin)
}

// Confirm prompts for a yes/no confirmation. Returns true if user enters "y" or "yes".
func Confirm(prompt string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	answer, err := readLine(os.Stdin)
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// Select presents a numbered list on stderr and returns the chosen index. A
// single option is returned immediately without prompting. Out-of-range or
// non-numeric input re-prompts. Prompts go to stderr so structured stdout
// output stays clean.
func Select(label string, options []string) (int, error) {
	switch len(options) {
	case 0:
		return 0, fmt.Errorf("no options to choose from")
	case 1:
		return 0, nil
	}
	fmt.Fprintln(os.Stderr, label)
	for i, opt := range options {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, opt)
	}
	for {
		fmt.Fprintf(os.Stderr, "Select [1-%d]: ", len(options))
		line, err := readLine(os.Stdin)
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintln(os.Stderr, "Please enter a number in range.")
			continue
		}
		return n - 1, nil
	}
}

func readLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no input received")
}
