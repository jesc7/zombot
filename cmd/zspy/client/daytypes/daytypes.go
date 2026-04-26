package daytypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jesc7/zombot/server/types"
)

type DayType int

const (
	DtWork DayType = iota //day type
	DtShort
	DtHoliday
)

type jobCal struct {
	Year   int `json:"year"`
	Months []struct {
		Month int    `json:"month"`
		Days  string `json:"days"`
	} `json:"months"`
}

var cal = make(map[string]jobCal)

func GetDayType(country string, t time.Time) (dt DayType, e error) {
	if cal[country].Year != t.Year() {
		cal[country] = jobCal{}
	}
	if len(cal[country].Months) == 0 {
		buf := new(bytes.Buffer)
		var (
			resp  *http.Response
			fname = types.Join(path.Dir(os.Args[0]), "daytypes", fmt.Sprintf("%s_%d.json", country, t.Year()))
		)

		fromFile := false
		if b, e := os.ReadFile(fname); e == nil {
			if _, e = buf.Write(b); e == nil {
				fromFile = true
			}
		}

		if !fromFile {
			client := http.Client{
				Transport: &http.Transport{},
				Timeout:   5 * time.Second,
			}
			resp, e = client.Get(fmt.Sprintf("http://xmlcalendar.ru/data/%s/%d/calendar.json", country, t.Year()))
			if e == nil {
				defer resp.Body.Close()
				if _, e = buf.ReadFrom(resp.Body); e != nil {
					return
				}
				if e = types.Str2File(buf.String(), fname); e != nil {
					return
				}
			} else {
				var b []byte
				if b, e = os.ReadFile(fname); e != nil {
					return
				}
				if _, e = buf.Write(b); e != nil {
					return
				}
			}
		}

		dec := json.NewDecoder(buf)
		c := jobCal{}
		if e = dec.Decode(&c); e != nil {
			return
		}
		cal[country] = c
	}
	for _, v := range cal[country].Months {
		if int(t.Month()) == v.Month {
			for _, d := range strings.Split(v.Days, ",") {
				if len(d) > 0 {
					sign := d[len(d)-1:]
					if sign == "+" || sign == "*" {
						d = d[:len(d)-1]
					}
					if strconv.Itoa(t.Day()) == d {
						if sign == "*" {
							return DtShort, nil
						}
						return DtHoliday, nil
					}
				}
			}
		}
	}
	return
}
