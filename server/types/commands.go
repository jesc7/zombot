package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jesc7/zombot/cmd/zspy/shared"
	"github.com/jesc7/zombot/server/types"
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
	days, _ := strconv.Atoi(m["days"])
	return b, days
}

func IsCommand(text string) bool {
	_command := func(text string) (string, bool) {
		if strings.Index(text, "/") == 0 {
			if strings.Contains(text, ":") {
				return strings.Split(text, ":")[0], true
			}
			return text, true
		}
		return "", false
	}
	_params := func(text string) string {
		if strings.Index(text, "/") == 0 {
			if strings.Contains(text, ":") {
				return strings.Split(text, ":")[1]
			}
			return ""
		}
		return ""
	}

	if isHelp(text) {
		text = "/help"
	} else if duty, name, days := isDuty(text); duty {
		text = fmt.Sprintf("/duty:%s#%d", name, days)
	} else if isAbsent(text) {
		text = "/absent"
	} else if bd, days := isBirthday(text); bd {
		text = fmt.Sprintf("/birthday:%d", days)
	}

	cmd, ok := _command(text)
	if !ok {
		return false
	}

	switch cmd {
	case "/help": //помощь
		b.SendText(MSG_HELP)

	case "/duty": //дежурства
		params := strings.Split(upd.GetParam(), "#")
		name, days := params[0], 7
		if len(params) > 1 {
			days, _ = strconv.Atoi(params[1])
		}
		env, e := shared.Pack(shared.TypeMessageDuties, shared.MessageDuties{
			Q: shared.DutyQuery{
				Name: name,
				Days: days,
			},
		})
		if e != nil {
			break
		}
		b.b.Write(types.BUS_WS, env)

	case "/absent": //отсутствующие
		env, e := shared.Pack(shared.TypeMessageAbsents, shared.MessageAbsents{})
		if e != nil {
			break
		}
		b.b.Write(types.BUS_WS, env)

	case "/birthday": //дни рождения
		days, _ := strconv.Atoi(upd.GetParam())
		if days <= 0 {
			days = 31
		}
		env, e := shared.Pack(shared.TypeMessageBirthdays, shared.MessageBirthdays{Days: days})
		if e != nil {
			break
		}
		b.b.Write(types.BUS_WS, env)

	case "/ci": //инфо о клиентах
	}

}
