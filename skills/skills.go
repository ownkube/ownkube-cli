// Package skills embeds the agent skills okctl installs with
// `okctl skills install`, so a binary always installs the skill that matches
// its own commands.
package skills

import _ "embed"

// Okctl is the okctl skill's SKILL.md.
//
//go:embed okctl/SKILL.md
var Okctl []byte
