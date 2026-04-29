package bot

import (
	"log"
	"regexp"
	"strconv"
)

var (
	reHelp     = regexp.MustCompile(`(?i)^помощь$`)
	reDuty     = regexp.MustCompile(`(?i)^дежур[а-я]*(?:(?:\s+(?P<name>[а-я]+))?(?:\s+(?P<days>\d+))?)?$`)
	reAbsent   = regexp.MustCompile(`(?i)отсутств[а-я]*`)
	reBirthday = regexp.MustCompile(`(?i)(?:день|дни) рожд[а-я]*(?:\s+(?P<days>\d+))?`)
)

func findCommand(re *regexp.Regexp, value string) (bool, map[string]string) {
	res := re.FindStringSubmatch(value)
	if res == nil {
		return false, nil
	}

	groups := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i < len(res) && i != 0 && name != "" {
			groups[name] = res[i]
		}
	}
	return true, groups
}

func isHelp(value string) bool {
	b, _ := findCommand(reHelp, value)
	return b
}

func isDuty(value string) (bool, string, int) {
	b, m := findCommand(reDuty, value)
	if !b {
		return b, "", 0
	}
	name := m["name"]
	days, _ := strconv.Atoi(m["days"])
	return b, name, days
}

func isAbsent(value string) bool {
	b, _ := findCommand(reAbsent, value)
	return b
}

func isBirthday(value string) (bool, int) {
	b, m := findCommand(reBirthday, value)
	if !b {
		return b, 0
	}

	log.Printf("%#v", m)

	days, _ := strconv.Atoi(m["days"])
	return b, days
}
