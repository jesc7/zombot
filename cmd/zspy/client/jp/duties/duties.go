package duties

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/daytypes"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type Planner map[time.Time]string

func DutiesList(ctx context.Context, db *sql.DB) (*Planner, error) {
	pl := make(Planner)
	rows, e := db.QueryContext(ctx, `
		select t.dt, list(u.username, ', ')
		from tabel t
		left join sp$users u on u.id = t.user_id
		where t.tabel_type = 5
		and t.dt between current_date and dateadd(day, 365, current_date)
		group by t.dt
		order by t.dt
	`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	t, s := time.Time{}, ""
	for rows.Next() {
		if e = rows.Scan(&t, &s); e != nil {
			return nil, e
		}
		pl[types.ClearTime(t)] = s
	}
	return &pl, nil
}

func Duty(ctx context.Context, db *sql.DB, q shared.DutyQuery) ([]shared.Daily, error) {
	pl, e := DutiesList(ctx, db)
	if e != nil {
		return nil, e
	}

	start := 1
	if time.Now().Hour() < 17 {
		start = 0
	}
	if q.Days <= 0 {
		q.Days = 7
	}

	var res []shared.Daily
	for i := start; i <= q.Days; i++ {
		t := types.ClearTime(time.Now()).Add(24 * time.Hour * time.Duration(i))
		if d, ok := (*pl)[t]; ok && (q.Name == "" || types.ContainsWord(d, q.Name)) {
			res = append(res, shared.Daily{
				Date:    t,
				Caption: d,
			})
		}
	}
	return res, nil
}

func MissDuties(ctx context.Context, db *sql.DB, days int) string {
	pl, e := DutiesList(ctx, db)
	if e != nil {
		return ""
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
		if dut, ok := (*pl)[t]; ok {
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

	if len(ds) == 0 {
		return ""
	}

	repl := strings.NewReplacer(
		"Jan", "января", "Feb", "февраля", "Mar", "марта", "Apr", "апреля", "May", "мая", "Jun", "июня",
		"Jul", "июля", "Aug", "августа", "Sep", "сентября", "Oct", "октября", "Nov", "ноября", "Dec", "декабря",
		"Mon", "понедельник", "Tue", "вторник", "Wed", "среда", "Thu", "четверг", "Fri", "пятница", "Sat", "суббота", "Sun", "воскресенье",
	)
	var res string
	res = "🤖 <b>Неплохо бы назначить дежурных:</b>"
	for _, v := range ds {
		res += repl.Replace(v.t.Format("\n_2 Jan (Mon)")) + v.s
	}
	return res
}

func TomorrowDuties(ctx context.Context, db *sql.DB) string {
	rows, e := db.QueryContext(ctx, `
		select t.dt, t.tabel_type, u.username, coalesce(p.gender, 0), coalesce(p.tg_id, 0), coalesce(p.sched_id, 0)
		from tabel t
		join sp$users u on t.user_id = u.id
		join u$personal p on t.user_id = p.user_id
		where 1 = 1
			and t.dt = dateadd(day, 1, current_date)
			and t.tabel_type in (6, 7)
		order by t.tabel_type
	`)
	if e != nil {
		return ""
	}
	defer rows.Close()

	var (
		res      string
		dt       time.Time
		ttype    int
		name     string
		gender   int
		tg_id    int
		sched_id int
	)
	for rows.Next() {
		if e = rows.Scan(&dt, &ttype, &name, &gender, &tg_id, &sched_id); e == nil {
			if gender < 0 || gender > 1 {
				gender = 0
			}
			res += fmt.Sprintf("%s - %s\n", types.Iif(ttype == 6, "🌒 Утро", "☀️ День"), strings.Trim(name, " "))
		}
	}
	if len(res) != 0 {
		res = "<b>👷 Дежурные на завтра</b>\n" + types.Iif(strings.Count(res, "\n") > 1, "\n", "") + res
	}
	return res
}

func HolidaysCount(ctx context.Context, db *sql.DB) int {
	pl, e := DutiesList(ctx, db)
	if e != nil {
		return 0
	}

	t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	if _, ok := (*pl)[t]; ok { //если мы уже внутри выходных, то не реагируем
		return 0
	}

	var res int
	for i := 1; i <= 20; i++ {
		if _, ok := (*pl)[t.AddDate(0, 0, i)]; !ok {
			break
		}
		res++
	}
	if res == 2 && t.AddDate(0, 0, 1).Weekday() == time.Saturday { //если впереди 2 выходных и завтра суббота, то не реагируем
		res = 0
	}
	return res
}
