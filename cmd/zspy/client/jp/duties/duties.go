package duties

import (
	"context"
	"database/sql"
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
