package planner

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/jp/duties"
	"github.com/jesc7/zombot/cmd/zspy/client/types"
)

/*
Ratings подготавливает различные motivation-рейтинги сотрудников за период
*/
func Ratings(ctx context.Context, db *sql.DB, weekly bool, start time.Time) string {
	pl, e := duties.DutiesList(ctx, db)
	if e != nil {
		return ""
	}

	type list struct {
		name  string
		value int
		j     int
	}

	/*
		_lates возвращает список опоздавших и величину опоздания
		minutes - период в минутах, который не считается опозданием
	*/
	_lates := func(t time.Time, minutes uint) []list {
		rows, e := db.Query(`
			select h.dt, u.username, datediff(second, sc.tfrom, h.time_in)
			from tabel t
			join tabel_history h on h.user_id = t.user_id and h.dt = t.dt
			join sp$users u on h.user_id = u.id
			left join pr_getsched_v2(0, u.id, h.dt) sc on 1 = 1
			where u.status <> -1 and h.comments_id = 1 and h.time_out is null
				and h.time_in between dateadd(minute, ?, sc.tfrom) and dateadd(minute, 60, sc.tfrom)
				and h.dt between ? and current_date
			order by 1,2
		`, minutes, t)
		if e != nil {
			return nil
		}
		defer rows.Close()

		var (
			dt    time.Time
			name  string
			value int
			m     = make(map[string]int)
		)
		for rows.Next() {
			if e = rows.Scan(&dt, &name, &value); e != nil {
				return nil
			}
			if _, ok := (*pl)[types.ClearTime(dt)]; !ok { //пропускаем дни с дежурствами
				m[name] += value
			}
		}
		var res []list
		for k, v := range m {
			res = append(res, list{name: k, value: v / 60})
		}
		sort.SliceStable(res, func(i, j int) bool { return res[i].value > res[j].value })
		return res
	}

	/*
		_continuous возвращает список сотрудников и признак опоздал/нет
		minutes - период в минутах, который не считается опозданием
	*/
	_continuous := func(t time.Time, cnt int, minutes uint) (r []list) {
		rows, e := db.Query(`
			select h.dt, u.username, iif(h.time_in <= dateadd(minute, ?, sc.tfrom), 0, 1)
			from tabel t
			join tabel_history h on h.user_id = t.user_id and h.dt = t.dt
			join sp$users u on h.user_id = u.id
			left join pr_getsched_v2(0, u.id, h.dt) sc on 1 = 1
			where u.status <> -1 and h.comments_id = 1 and h.time_out is null and h.dt between ? and current_date
		`, minutes, t)
		if e != nil {
			return
		}
		defer rows.Close()

		var (
			dt    time.Time
			name  string
			value int
			m     = make(map[string][2]int)
		)
		for rows.Next() {
			if e = rows.Scan(&dt, &name, &value); e != nil {
				return
			}
			if _, ok := pl[time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, time.Local)]; !ok { //пропускаем дни с дежурствами
				v := m[name]
				v[0] += value
				v[1]++
				m[name] = v
			}
		}
		var tmp []list
		for k, v := range m {
			if v[1] > 2 {
				tmp = append(tmp, list{name: k, value: v[0], j: v[1]})
			}
		}

		if cnt < 0 {
			for i := range tmp {
				tmp[i].value = tmp[i].j - tmp[i].value
			}
			cnt = -cnt
		}
		sort.SliceStable(tmp, func(a, b int) bool {
			return tmp[a].value < tmp[b].value || (tmp[a].value == tmp[b].value && tmp[a].j > tmp[b].j)
		})

		for _, v := range tmp {
			if v.value < cnt {
				if len(r) == 0 || r[len(r)-1].value != v.value {
					r = append(r, v)
				} else {
					r[len(r)-1].name += ", " + v.name
				}
			}
		}
		return r
	}

	/*
		_maxWorker возвращает список сотрудников и величину переработки в минутах согласно рабочего расписания сотрудника
	*/
	_maxWorker := func(t time.Time) (r []list) {
		rows, e := db.Query(`
			select a.dt, a.username, sum(datediff(minute, a.tin, a.tout) - datediff(minute, a.tfrom, a.tto))
			from (
				select min(h.time_in) as tin, max(h.time_out) as tout, h.dt, u.username, s.tfrom, s.tto
				from tabel_history h
				join sp$users u on h.user_id = u.id
				left join pr_getsched_v2(0, h.user_id, h.dt) s on 1 = 1
				where u.status <> -1 and h.comments_id in (1, 2) and h.dt between ? and current_date
				group by h.dt, h.user_id, u.username, s.tfrom, s.tto
			) a
			where not a.tin is null and not a.tout is null
			group by 1, 2
			order by 1 desc
		`, t)
		if e != nil {
			return
		}
		defer rows.Close()

		var (
			dt    time.Time
			name  string
			value int
			m     = make(map[string]int)
		)
		for rows.Next() {
			if e = rows.Scan(&dt, &name, &value); e != nil {
				return
			}
			if _, ok := pl[time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, time.Local)]; !ok { //пропускаем дни с дежурствами
				m[name] += value
			}
		}

		n := make(map[int][]string)
		for k, v := range m {
			item := n[v]
			item = append(item, k)
			n[v] = item
		}
		for k, v := range n {
			sort.SliceStable(v, func(i, j int) bool { return v[i] < v[j] })
			r = append(r, list{name: strings.Join(v, ", "), value: k})
		}
		sort.SliceStable(r, func(i, j int) bool { return r[i].value > r[j].value })
		return
	}

	/*
		_edited возвращает список сотрудников и число исправлений времени начала работы в табеле
		minutes - период в минутах, который не считается за исправление
	*/
	_edited := func(t time.Time, minutes uint) (r []list) {
		rows, e := db.Query(`
			select b.dt, b.username, sum(b.d1)
			from (
				select a.dt, a.username, iif(abs(datediff(minute, t, coalesce(tin, t))) > ?,1,0) as d1, iif(abs(datediff(minute, t, coalesce(tout, t))) > 1,1,0) as d2
				from (
					select h.dt, u.username, cast(h.insertdt as time) as t, cast(h.time_in as time) as tin, cast(h.time_out as time) as tout
					from tabel_history h
					join sp$users u on h.user_id = u.id
					where u.status <> -1 and (((h.comments_id = 1) and (h.time_out is null)) or ((h.comments_id = 2) and (h.time_in is null)))	and h.dt between ? and current_date
					order by h.dt desc
				) a
			) b
			where (b.d1 = 1)
			group by 1,2
			order by 1 desc
		`, minutes, t)
		if e != nil {
			return
		}
		defer rows.Close()

		var (
			dt    time.Time
			name  string
			value int
			m     = make(map[string]int)
		)
		for rows.Next() {
			if e = rows.Scan(&dt, &name, &value); e != nil {
				return
			}
			if _, ok := pl[time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, time.Local)]; !ok { //пропускаем дни с дежурствами
				m[name] += value
			}
		}

		n := make(map[int][]string)
		for k, v := range m {
			item := n[v]
			item = append(item, k)
			n[v] = item
		}
		for k, v := range n {
			sort.SliceStable(v, func(i, j int) bool { return v[i] < v[j] })
			r = append(r, list{name: strings.Join(v, ", "), value: k})
		}
		sort.SliceStable(r, func(i, j int) bool { return r[i].value > r[j].value })
		return
	}

	res = funcs.Iif(weekly == 0, "<b>Рейтинг 'Неделька'</b>", "<b>Победители по итогам месяца</b> 🥁🥁🥁")
	r := _continuous(start, 5, 5) //опоздания до 5 минут не считаются
	if len(r) != 0 {
		var s string
		switch r[0].value {
		case 0:
			s = "без опозданий"
		case 1:
			s = "1 опоздание"
		case 2, 3, 4:
			s = strconv.Itoa(r[0].value) + " опоздания"
		case 5:
			s = "5 опозданий"
		}
		if len(s) != 0 {
			res += fmt.Sprintf("\n\n🏆 <b>Номинация 'Человек-загадка'</b>\n%s (%s)", r[0].name, s)
		}
	}

	r = _continuous(start, -5, 5) //опоздания до 5 минут не считаются
	if len(r) != 0 {
		var s string
		switch r[0].value {
		case 0:
			s = "опоздал(а) вообще везде"
		case 1:
			s = "1 день без опозданий"
		case 2, 3, 4:
			s = strconv.Itoa(r[0].value) + " дня без опозданий"
		case 5:
			s = "5 дней без опозданий"
		}
		if len(s) != 0 {
			res += fmt.Sprintf("\n\n🏆 <b>Номинация 'Слава богу, ты пришел'</b>\n%s (%s)", r[0].name, s)
		}
	}

	r = _lates(start, 5) //опоздания до 5 минут не считаются
	if len(r) != 0 {
		res += fmt.Sprintf("\n\n🏆 <b>Номинация 'Засоня %s'</b>\n%s (%d мин. опозданий)", funcs.Iif(weekly == 0, "недели", "месяца"), r[0].name, r[0].value)
	}

	r = _maxWorker(start) //величина переработки в минутах
	if len(r) != 0 {
		res += fmt.Sprintf("\n\n🏆 <b>Номинация 'Переработник %s'</b>\n%s (%+d мин.)", funcs.Iif(weekly == 0, "недели", "месяца"), r[0].name, r[0].value)
	}

	r = _edited(start, 5) //гэп 5 минут не считается за редактирование
	if len(r) != 0 {
		res += fmt.Sprintf("\n\n🏆 <b>Номинация 'Гений фотошопа'</b>\n%s (%d исправлений)", r[0].name, r[0].value)
	}
	return
}
