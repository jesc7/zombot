package bot

import "regexp"

var (
	reDuty = regexp.MustCompile(`(?i)^дежур[а-я]*(?:(?:\s+(?P<name>[а-я]+))?(?:\s+(?P<days>\d+))?)?$`)
)

func isCommand(re *regexp.Regexp, value string) (bool, *map[string]string) {
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
