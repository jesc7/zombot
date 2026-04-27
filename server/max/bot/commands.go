package bot

import (
	"regexp"
	"strconv"
)

var (
	reDuty = regexp.MustCompile(`(?i)^дежур[а-я]*(?:(?:\s+(?P<name>[а-я]+))?(?:\s+(?P<days>\d+))?)?$`)
)

func findCommand(re *regexp.Regexp, value string) (bool, *map[string]string) {
	res := re.FindStringSubmatch(value)
	if res == nil {
		return false, nil
	}

	groups := make(map[string]string)
	for i, name := range reDuty.SubexpNames() {
		if i != 0 && name != "" {
			groups[name] = res[i]
		}
	}
	return true, &groups
}

func isDuty(value string) (bool, string, int) {
	b, m := findCommand(reDuty, value)
	if !b {
		return b, "", 0
	}
	name := (*m)["name"]
	days, _ := strconv.Atoi((*m)["name"])
}
