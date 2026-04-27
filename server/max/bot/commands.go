package bot

import "regexp"

var (
	reDuty = regexp.MustCompile(`(?i)^дежур[а-я]*(?:(?:\s+(?P<name>[а-я]+))?(?:\s+(?P<days>\d+))?)?$`)
)

func isDutyCommand(value string) (bool, string, int) {
	reDuty.FindAllString()