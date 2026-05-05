package planner

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/daytypes"
	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
	"github.com/jesc7/zombot/cmd/zspy/shared"
)

// Absents возвращает список отсутствующих и причину отсутствия
func Absents(ctx context.Context, db *sql.DB) ([]shared.Absent, error) {
	pl, e := duties.DutiesList(ctx, db)
	if e != nil {
		return nil, e
	}
	if _, ok := (*pl)[types.ClearTime(time.Now())]; ok { //если сегодня есть дежурные, значит нерабочий день, не проверяем отсутствующих
		return []shared.Absent{}, nil
	}

	rows, e := db.QueryContext(ctx, `
		select t1, t2, u, lower(trim(iif(t1 = 0, c2, iif((t1 = 6) or (t1 = 7), c1 || iif(char_length(c1) > 0 and char_length(c2) > 0, ' / ', '') || c2, c1)))) as c, g
		from (
			select coalesce(t.tabel_type, -1) as t1, coalesce(tc.id, -1) as t2, u.username as u, coalesce(tt.caption, '') as c1, coalesce(tc.caption, '') as c2, coalesce(p.gender, 0) as g
			from tabel t
			join sp$users u on u.id = t.user_id
			join u$personal p on u.id = p.user_id
			left join vw_tabel_type tt on t.tabel_type = tt.id
			left join (
				select user_id, max(id) as id from tabel_history where dt = current_date and comments_id <> 2 group by 1
			) h on u.id = h.user_id
			left join tabel_history th on h.id = th.id
			left join tabel_history_comments tc on th.comments_id = tc.id
			where u.status = 0 and not u.id in (-1, 0, 8, 82) and t.dt = current_date and th.time_in is null
		) a
		order by u
	`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	var (
		res                  []shared.Absent
		type1, type2, gender int
		name, comment        string
	)
	for rows.Next() {
		if e = rows.Scan(&type1, &type2, &name, &comment, &gender); e == nil {
			if gender < 0 || gender > 1 {
				gender = 0
			}
			a := shared.Absent{
				Name:    name,
				Gender:  shared.GenderType(gender),
				Comment: comment,
			}

			switch type1 {
			case -1:
				a.Type = shared.AT_DUNNO
			case 2:
				a.Type = shared.AT_ILL
			case 3:
				a.Type = shared.AT_LEAVE
			case 6, 7: //поправил дежурных, проверить
				if type2 == -1 {
					type2 = 5
				}
				fallthrough
			default:
				switch type2 {
				case 3:
					a.Type = shared.AT_DINNER
				case 4:
					a.Type = shared.AT_OFF
				case 5:
					a.Type = shared.AT_WORK
				default:
					a.Type = shared.AT_DUNNO
				}
			}
			res = append(res, a)
		}
	}
	return res, nil
}

// Birthdays возвращает дни рождения сотрудников
func Birthdays(ctx context.Context, db *sql.DB, days int) ([]shared.Birthday, error) {
	rows, e := db.QueryContext(ctx, `
		select *
		from (
			select iif(b.dr < current_date, dateadd(year, 1, b.dr), b.dr) as dr, b.caption, b.g
			from (
				select cast(extract(year from current_date)||'-'||extract(month from dr)||'-'||extract(day from dr) as date) as dr, a.caption, a.g
				from (
					select dr, caption, coalesce(gender, 0) as g
					from u$personal
					where status = 0 and folder_id = 93 and not dr is null
					union
					select cast('31.07.1980' as date) as dr, 'Гарри Поттер' as caption, 1 as g from rdb$database
				) a
			) b
		) c
		where c.dr between current_date and dateadd(day, ?, current_date)
		order by 1,2
		`, days-1)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	var res []shared.Birthday
	for rows.Next() {
		d, s, g := time.Time{}, "", 0
		if e = rows.Scan(&d, &s, &g); e != nil {
			continue
		}
		res = append(res, shared.Birthday{
			Daily: shared.Daily{
				Date:    types.ClearTime(d),
				Caption: s,
			},
			Gender: shared.GenderType(g),
		})
	}
	return res, nil
}

func CriticalTasks(ctx context.Context, db *sql.DB, minutes int) string {
	rows, e := db.QueryContext(ctx, `
		select r.id, r.insertdt, r.point_id, p.caption, r.atext, datediff(minute, r.insertdt, current_timestamp), r.clientinfo
		from srq$requests r
		join sp$group_detail gd on gd.grouptable_id = r.id
		join points p on p.id = r.point_id
		where (r.uchet_id = 0 or r.uchet_id is null)
			and r.sdeleted is null
			and gd.grouptable = 'SRQ$REQUESTS' and gd.group_id = 288
			and datediff(minute, r.insertdt, current_timestamp) > ?
			and datediff(minute, r.insertdt, current_timestamp) < ?
			order by r.insertdt desc
	`, minutes, minutes*2)
	if e != nil {
		return ""
	}
	defer rows.Close()

	var (
		id, point_id, dur              int64
		t                              time.Time
		res, caption, text, clientInfo string
	)
	for rows.Next() {
		if e = rows.Scan(&id, &t, &point_id, &caption, &text, &dur, &clientInfo); e != nil {
			continue
		}
		if len(clientInfo) != 0 {
			clientInfo = " (" + clientInfo + ")"
		}
		res = fmt.Sprintf("%s\n\n#<b>%d</b> (%d мин.) %s%s\n%s", res, id, dur, caption, clientInfo, text)
	}
	if len(res) != 0 {
		res = "⚡ <b>Срочные заявки</b>" + res
	}
	return res
}

type notes map[string]int

var lastEOW, lastSOW notes

// SowList (StartOfWork list) сообщает, что дежурный начал работу
func SowList(ctx context.Context, db *sql.DB, cwd string) string {
	dt, e := daytypes.GetDayType(cwd, "ru", time.Now())
	if e != nil || dt != daytypes.DtHoliday || !types.NowBetween("7:30", "9:30") {
		return ""
	}

	rows, e := db.QueryContext(ctx, fmt.Sprintf(`
		select u.username, h.time_in, coalesce(p.gender, 0) as g
		from tabel_history h
		join sp$users u on h.user_id = u.id
		join u$personal p on u.id = p.user_id
		where 1=1
			and h.comments_id = 1 
			and h.dt = current_date	
			and datediff(minute, h.time_in, time '%s') < 5
	`, time.Now().Format("15:04")))
	if e != nil {
		return ""
	}
	defer rows.Close()

	p, user, t, g := notes{}, "", time.Time{}, 0
	for rows.Next() {
		if e = rows.Scan(&user, &t, &g); e != nil {
			return ""
		}
		p[fmt.Sprintf("%s (%s)", user, t.Format("15:04"))] = g
	}

	var res string
	for k, v := range p {
		if _, ok := lastSOW[k]; !ok {
			res += types.RndFrom([2][]string{{"👩", "👩🏻", "👩🏼", "👩🏽"}, {"🧑", "🧑🏻", "🧑🏼", "🧑🏽"}}[v]...) + " " + k + "\n"
		}
	}
	lastSOW = p

	if res == "" {
		return ""
	}
	return fmt.Sprintf("<b>Я на месте</b>\n%s", res)
}

type eow struct {
	eot    time.Time
	name   string
	gender int
}

var eowList []eow

func EowClear() {
	eowList = []eow{}
}

// EowList (EndOfWork list) выводит список сотрудников, окончивших работу ДО окончания рабочего дня согласно рабочего расписания
func EowList(ctx context.Context, db *sql.DB, cnt int) string {
	_getPhrase := func(g int) string {
		phrases := []string{
			"Алоха",
			"Адиос мучачос",
			"Аста ла виста",
			"Арривидерчи",
			"Саёнара",
			"Сау болындар",
			"Всем пока",
			"Бай бай",
			"Чао рагацци",
			"Баюшки",
			"До завтра",
			"Гудбайте",
			"Покедова",
		}
		/*phrasesMale := []string{
			"Всё, я ушел",
			"Ушёл, всем пока",
			"Пора валить",
			"Досвидос",
			"Я устал, я ухожу",
			"Давай до свидания",
		}
		phrasesFemale := []string{
			"Ой, всё",
			"Пошла я",
			"Я ушла",
			"Оревуар",
			"Покасики",
			"Досвидули",
			"Я побежала",
		}
		switch g {
		case 0:
			phrases = append(phrasesFemale, phrases...)
		default:
			phrases = append(phrasesMale, phrases...)
		}*/

		phrasesByGender := [][]string{{
			"Ой, всё",
			"Пошла я",
			"Я ушла",
			"Оревуар",
			"Покасики",
			"Досвидули",
			"Я побежала",
		}, {
			"Всё, я ушел",
			"Ушёл, всем пока",
			"Пора валить",
			"Досвидос",
			"Я устал, я ухожу",
			"Давай до свидания",
		}}
		phrases = append(phrases, phrasesByGender[g]...)

		switch types.Rnd(0, 100) < 50 {
		case true:
			return phrases[0]
		default:
			return types.RndFrom(phrases[1:]...)
		}
	}

	rows, e := db.QueryContext(ctx, fmt.Sprintf(`
		select u.username, h.time_out, coalesce(p.gender, 0) as g
		from tabel_history h
		join tabel t on h.user_id = t.user_id and h.dt = t.dt
		join sp$users u on h.user_id = u.id
		join u$personal p on u.id = p.user_id
		left join pr_getsched_v2(0, u.id, null) a on 1 = 1
		where 1=1
		  and h.comments_id = 2 
		  and h.dt = current_date 
		  and h.time_out < a.tto 
		  --and datediff(second, h.time_out, time '%s') < 50
		order by 2, 1
	`, time.Now().Format("15:04")))
	if e != nil {
		return ""
	}
	defer rows.Close()

	p, user, t, g := notes{}, "", time.Time{}, 0
	_ = p
	var eowCurr []eow
	for rows.Next() {
		if e = rows.Scan(&user, &t, &g); e != nil {
			return ""
		}
		//p[fmt.Sprintf("%s (%s)", user, t.Format("15:04"))] = g
		eowCurr = append(eowCurr, eow{t, user, g})
	}
	res := ""
	for _, v := range eowCurr[len(eowList):] {
		res += fmt.Sprintf("%s %s (%s)\n", types.RndFrom([2][]string{{"🚶‍♀️", "🏃‍♀️", "🙋‍♀️"}, {"🚶🏻‍♂️", "🏃‍♂️", "🙋‍♂️"}}[v.gender]...), v.name, v.eot.Format("15:04"))
	}
	eowList = eowCurr

	/*res := ""
	for k, v := range p {
		if _, ok := lastEOW[k]; !ok {
			res += types.RndFrom([2][]string{{"🚶‍♀️", "🏃‍♀️", "🙋‍♀️"}, {"🚶🏻‍♂️", "🏃‍♂️", "🙋‍♂️"}}[v]...) + " " + k + "\n"
		}
	}
	lastEOW = p*/
	if len(res) != 0 {
		res = fmt.Sprintf("<b>%s</b>\n\n%s", _getPhrase(g), res)
	}
	return res
}

// ForeignHoliday проверяет, что клиенты в других странах сегодня отдыхают, в то время как мы работаем :(
func ForeignHoliday(cwd string) string {
	t := time.Now()
	dt, e := daytypes.GetDayType(cwd, "ru", t)
	if e != nil || dt == daytypes.DtHoliday {
		return ""
	}

	dt, e = daytypes.GetDayType(cwd, "kz", t)
	if e != nil || dt != daytypes.DtHoliday {
		return ""
	}

	return "✨ В KZ сегодня " + types.RndFrom("нерабочий день", "отдыхают", "что-то празднуют", "выходной", "праздник какой-то")
}
