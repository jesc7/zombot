package planner

import (
	"context"
	"database/sql"

	"github.com/jesc7/zombot/cmd/zspy/shared"
)

// Absents возвращает список отсутствующих и причину отсутствия
func Absents(ctx context.Context, db *sql.DB) ([]shared.Absent, error) {
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

	type abs struct {
		type1   int
		type2   int
		name    string
		comment string
		gender  int
	}

	var (
		res  []shared.Absent
		user abs
	)
	for rows.Next() {
		if e = rows.Scan(&user.type1, &user.type2, &user.name, &user.comment, &user.gender); e == nil {
			if user.gender < 0 || user.gender > 1 {
				user.gender = 0
			}
			a := shared.Absent{
				Name:    user.name,
				Gender:  shared.Gender(user.gender),
				Comment: user.comment,
			}

			switch user.type1 {
			case -1:
				a.Type = shared.AT_DUNNO
			case 2:
				a.Type = shared.AT_ILL
			case 3:
				a.Type = shared.AT_LEAVE
			case 6, 7: //поправил дежурных, проверить
				if user.type2 == -1 {
					user.type2 = 5
				}
				fallthrough
			default:
				switch user.type2 {
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
