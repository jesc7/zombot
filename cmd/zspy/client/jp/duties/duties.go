package duties

import (
	"context"
	"database/sql"
	"time"

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

	for rows.Next() {
		var (
			t time.Time
			s string
		)
		if e = rows.Scan(&t, &s); e != nil {
			return nil, e
		}
		pl[t] = s
	}
	return &pl, nil
}

func Duty(ctx context.Context, db *sql.DB, q shared.DutyQuery) ([]shared.Duty, error) {
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

	var res []shared.Duty
	for i := start; i <= q.Days; i++ {
		t := time.Now().Truncate(24 * time.Hour)
		if d, ok := (*pl)[t]; ok && (q.Name == "" || types.ContainsWord(d, q.Name)) {
			res = append(res, shared.Duty{
				Date: t,
				Name: d,
			})
		}
	}
	return res, nil
}
