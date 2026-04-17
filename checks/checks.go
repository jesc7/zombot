package checks

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jesc7/zombot/types"
	_ "github.com/nakagami/firebirdsql"
)

// checkResources check urls pool (GET requests only)
func CheckResources(sl []string) string {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	b := strings.Builder{}
	wg := &sync.WaitGroup{}
	for _, v := range sl {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			var (
				count = 3
				e     error
			)
			for i := 0; i < count; i++ {
				if _, e = client.Get(url); e == nil {
					break
				}
				log.Printf("check URL '%s' error: %v", url, e)
				time.Sleep(5 * time.Second)
			}
			if e != nil {
				b.WriteString("\n" + url)
			}
		}(v)
	}
	wg.Wait()
	if b.Len() != 0 {
		return "⚠️ <b>Ошибка проверки URL</b>" + b.String()
	}
	return ""
}

// checkCFResources check cf resources
func CheckCFResources(sl []string) string {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 20 * time.Second,
	}

	b := strings.Builder{}
	wg := &sync.WaitGroup{}
	for _, v := range sl {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			if e := func() error {
				resp, e := client.Get(url)
				if e != nil {
					return e
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return errors.New(resp.Status)
				}

				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				if len(buf.String()) != 0 {
					return errors.New(buf.String())
				}
				return nil
			}(); e != nil {
				log.Printf("check URL %s error: %v", url, e)
				b.WriteString(fmt.Sprintf("\n%s: %v", url, e))
			}
		}(v)
	}
	wg.Wait()
	if b.Len() != 0 {
		return "⚠️ <b>Ошибка проверки URL</b>" + b.String()
	}
	return ""
}

// WatchZsrv проверяет, запущены ли zsrv площадок
// посылает http-запрос на ресурс /ping, ожидает в ответе текст "pong"
func WatchZsrv(watchers []types.ZSrvWatch) string {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	b := strings.Builder{}
	wg := &sync.WaitGroup{}
	for _, v := range watchers {
		wg.Add(1)
		go func(w types.ZSrvWatch) {
			var e error
			defer func() {
				if e != nil {
					log.Printf("%s (%s) error: %v", w.Url, w.Caption, e)
					b.WriteString(w.Caption + "\n")
				}
				wg.Done()
			}()

			w.Url, _ = strings.CutSuffix(w.Url, "/")
			resp, e := client.Get(w.Url + "/ping")
			if e != nil {
				return
			}
			defer resp.Body.Close()

			buf, e := io.ReadAll(resp.Body)
			if e != nil {
				return
			}
			if string(buf) != "pong" {
				e = errors.New("wrong ping answer: " + string(buf))
				return
			}
		}(v)
	}
	wg.Wait()
	res := b.String()
	b.Reset()
	if len(res) != 0 {
		s := "⚠️ <b>Ошибка проверки площадок ОЗ</b>\n" + types.Iif(strings.Count(res, "\n") > 1, "\n", "") + res
		return s
	}
	return ""
}

// CheckWhois check WhoIs info by domain names
func CheckWhois(sl []string, days int) string {
	const (
		url_ = "https://api.whois.vu/?q=%s&clean"
		Day  = 24 * time.Hour
	)
	type T_api_whois_vu struct {
		Domain    string   `json:"domain"`
		Available string   `json:"available"`
		Type      string   `json:"type"`
		Registrar string   `json:"registrar"`
		Statuses  []string `json:"statuses"`
		Created   int64    `json:"created"`
		Expires   int64    `json:"expires"`
		Deletion  int64    `json:"deletion"`
		Whois     string   `json:"whois"`
	}

	const (
		url = "https://who-dat.as93.net/%s"
	)
	type T_who__dat_as93_net struct {
		Domain struct {
			Domain         string   `json:"domain"`
			Status         []string `json:"status"`
			CreatedDate    string   `json:"created_date"`
			ExpirationDate string   `json:"expiration_date"`
		} `json:"domain"`
		Registrar struct {
			Name string `json:"name"`
		} `json:"registrar"`
		Registrant struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"registrant"`
	}

	_parse := func(value string) (t time.Time, e error) {
		layouts := []string{time.RFC3339, time.DateTime}
		value = strings.Split(value, " (GMT")[0]
		for _, v := range layouts {
			if t, e = time.Parse(v, value); e == nil {
				return
			}
		}
		return
	}

	b := strings.Builder{}
	wg := &sync.WaitGroup{}
	for _, v := range sl {
		wg.Add(1)
		go func(domain string) {
			var e error
			defer func() {
				if e != nil {
					log.Printf("check WhoIs error: %s - %v", domain, e)
				}

				wg.Done()
			}()

			buf := new(bytes.Buffer)
			if e = func(b *bytes.Buffer) error {
				client := &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
					Timeout: 10 * time.Second,
				}

				resp, e := client.Get(fmt.Sprintf(url, domain))
				if e != nil {
					return e
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return errors.New(resp.Status)
				}
				_, e = b.ReadFrom(resp.Body)
				return e
			}(buf); e != nil {
				return
			}

			ans := T_who__dat_as93_net{}
			if e = json.NewDecoder(buf).Decode(&ans); e != nil {
				return
			}

			exp, e := _parse(ans.Domain.ExpirationDate)
			if e != nil {
				if exp, e = _parse(ans.Domain.CreatedDate); e != nil {
					return
				}
				exp = time.Date(time.Now().Year(), exp.Month(), exp.Day(), 0, 0, 0, 0, time.Local)
				if exp.Before(time.Now()) {
					exp = time.Date(exp.Year()+1, exp.Month(), exp.Day(), 0, 0, 0, 0, time.Local)
				}
			}

			switch d := int(time.Until(exp) / Day); {
			case d >= -3 && d <= days:
				b.WriteString(fmt.Sprintf("\n%s: %s (%d дн)%s", domain, exp.Format("02.01.2006"), d,
					types.Iif(len(ans.Registrant.Country) != 0, " "+ans.Registrant.Name, "")))
			default:
			}
		}(v)
	}
	wg.Wait()
	if s := b.String(); len(s) != 0 {
		return "⚠️ <b>Срок регистрации домена заканчивается</b>" + types.Iif(strings.Count(s, "\n") > 1, "\n", "") + s
	}
	return ""
}

type EC struct {
	Driver  string `json:"driver"`
	ConnStr string `json:"connstr"`
	Pwd     string `json:"pwd"`
}

func CheckEC(ec EC) string {
	b, _ := base64.StdEncoding.DecodeString(ec.Pwd)
	db, e := sql.Open(ec.Driver, fmt.Sprintf(ec.ConnStr, string(b)))
	if e != nil {
		log.Printf("CheckEC error: %v", e)
		return ""
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cnt int
	if e = db.QueryRowContext(ctx, `
		select count(1)
		from EC_STATIONS
		where last_comp_info starting with 'WARNING_DISKSPACE'
			and updatedt > dateadd(day, -7, current_date)
			and not exists (
				select id from sp$group_detail g where g.group_id = 25 and g.grouptable_id = pcid
			)
	`).Scan(&cnt); e != nil {
		log.Printf("CheckEC error: %v", e)
		return ""
	}

	if cnt != 0 {
		return "⚠️ <b>Место на диске заканчивается</b>\nПроблемных точек: " + strconv.Itoa(cnt)
	}
	return ""
}
