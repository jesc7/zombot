package duties

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/daytypes"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
)

type Planner map[time.Time]string

var (
	lastDuties, CurDuties Planner
)

func DutiesList(db *sql.DB) (res Planner, delta string) {
	res = make(Planner)
	rows, e := db.Query(`
		select t.dt, list(u.username, ', ')
		from tabel t
		left join sp$users u on u.id = t.user_id
		where t.tabel_type = 5
		and t.dt between current_date and dateadd(day, 365, current_date)
		group by t.dt
		order by t.dt
	`)
	if e != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			t time.Time
			s string
		)
		if e = rows.Scan(&t, &s); e != nil {
			return
		}
		res[t] = s
	}

	if len(res) == 0 {
		return
	}

	lastDuties = CurDuties
	CurDuties = res
	if lastDuties != nil {
		for i := 1; i <= 100; i++ {
			daytip := ""
			if i <= 3 {
				daytip = []string{" (сегодня)", " (завтра)", " (послезавтра)", " (через 2 дня)"}[i]
			}
			t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+i, 0, 0, 0, 0, time.Local)
			e1, ok1 := lastDuties[t]
			e2, ok2 := CurDuties[t]
			switch {
			case !ok1 && ok2: //новое дежурство
				delta += fmt.Sprintf("⭐ %s%s: %s\n", t.Format("02.01"), daytip, e2)
			case ok1 && !ok2: //отмена
				delta += fmt.Sprintf("🚫 %s%s: %s\n", t.Format("02.01"), daytip, e1)
			case ok1 && ok2 && (e1 != e2): //замена
				delta += fmt.Sprintf("🔄 %s%s: %s\n", t.Format("02.01"), daytip, e2)
			}
		}
		if len(delta) != 0 {
			delta = "👷 <b>Изменения дежурств</b>\n" + types.Iif(strings.Count(delta, "\n") > 1, "\n", "") + delta
		}
	}
	return
}

func Duties(db *sql.DB, daysCount int, dut Planner, who string) string {
	d := daysCount
	switch i := len(who); i {
	case 0:
	case 1, 2:
		who = ""
	default:
		who = types.Words(who, "")[0]
		daysCount = 365
	}
	if dut == nil {
		dut, _ = DutiesList(db)
	}
	start := 1
	if time.Now().Hour() < 17 {
		start = 0
	}
	var res, resWho string
	for i := start; i <= daysCount; i++ {

		time.Now().Truncate(24 * time.Hour)

		t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+i, 0, 0, 0, 0, time.Local)
		if empl, ok := dut[t]; ok {
			daytip := ""
			if i <= 3 {
				daytip = []string{" (сегодня)", " (завтра)", " (послезавтра)", " (через 2 дня)"}[i]
			}
			s := fmt.Sprintf("%s%s: %s\n", t.Format("02.01"), daytip, empl)
			if i <= d {
				res += s
			}
			if types.ContainsWord(empl, who) {
				resWho += s
			}
		}
	}
	if len(res+resWho) != 0 {
		switch who {
		case "":
			res = fmt.Sprintf("👷 <b>Дежурные</b>\n%s", types.Iif(strings.Count(res, "\n") > 1, "\n", "")+res)
		default:
			if resWho != "" {
				res = fmt.Sprintf("👷 <b>%s дежурит:</b>\n%s", who, types.Iif(strings.Count(resWho, "\n") > 1, "\n", "")+resWho)
			} else {
				res = fmt.Sprintf("👷 <b>%s хз когда дежурит, а вообще вот:</b>\n%s", who, types.Iif(strings.Count(res, "\n") > 1, "\n", "")+res)
			}
		}
	}
	return res
}

func HolidaysCount(db *sql.DB) (res int) {
	dut, _ := DutiesList(db)
	t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	if _, ok := dut[t]; ok { //если мы уже внутри выходных, то не реагируем
		return
	}
	for i := 1; i <= 20; i++ {
		if _, ok := dut[t.AddDate(0, 0, i)]; !ok {
			break
		}
		res++
	}
	if res == 2 && t.AddDate(0, 0, 1).Weekday() == time.Saturday { //если впереди 2 выходных и завтра суббота, то не реагируем
		res = 0
	}
	return
}

func MissDuties(db *sql.DB, days int) (res string) {
	dr := strings.NewReplacer(
		"Jan", "января", "Feb", "февраля", "Mar", "марта", "Apr", "апреля", "May", "мая", "Jun", "июня",
		"Jul", "июля", "Aug", "августа", "Sep", "сентября", "Oct", "октября", "Nov", "ноября", "Dec", "декабря",
		"Mon", "понедельник", "Tue", "вторник", "Wed", "среда", "Thu", "четверг", "Fri", "пятница", "Sat", "суббота", "Sun", "воскресенье",
	)

	if CurDuties == nil {
		DutiesList(db)
	}

	type needs struct {
		t time.Time
		s string
	}

	now, ds := time.Now(), []needs{}
	for i := 1; ; i++ {
		t := time.Date(now.Year(), now.Month(), now.Day()+i, 0, 0, 0, 0, time.Local)
		count, dutCount, countries := 0, 0, []string{}

	out:
		for i, v := range []string{"ru", "kz"} { //by, ua, uz
			dt, _ := daytypes.GetDayType(v, t)
			switch i == 0 {
			case true:
				if dt != daytypes.DtHoliday {
					break out
				}
				count++
			default:
				if dt != daytypes.DtHoliday {
					count++
					countries = append(countries, strings.ToUpper(v))
				}
			}
		}
		if count == 0 {
			if i > days {
				break
			}
			continue
		}
		if dut, ok := CurDuties[t]; ok {
			dutCount = strings.Count(dut, ",") + 1
		}
		if dutCount < count {
			switch dutCount {
			case 0:
				ds = append(ds, needs{t, types.Iif(count == 1, "", " - 2 чел, работают "+strings.Join(countries, ","))})
			default:
				ds = append(ds, needs{t, types.Iif(count == 1, "", " - доп.дежурный, работают "+strings.Join(countries, ","))})
			}
		}
	}
	if len(ds) != 0 {
		res = "🤖 <b>Неплохо бы назначить дежурных:</b>"
		for _, v := range ds {
			res += dr.Replace(v.t.Format("\n_2 Jan (Mon)")) + v.s
		}
	}
	return
}
