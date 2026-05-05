package types

import (
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
)

type Config struct {
	Host   string `json:"host"`
	Token  string `json:"token"`
	WebAPI struct {
		Port int `json:"port"`
	} `json:"webapi"`
	DB struct {
		Driver  string `json:"driver"`
		ConnStr string `json:"connstr"`
	} `json:"db"`
	CheckDomains []string    `json:"check_domains"`
	Checks       []string    `json:"checks"`
	CFChecks     []string    `json:"cf_checks"`
	ZSrv         []ZSrvWatch `json:"zsrv"`
	EC           EC          `json:"ec"`
}

type ZSrvWatch struct {
	Url     string `json:"url"`
	Caption string `json:"caption"`
}

type EC struct {
	Driver  string `json:"driver"`
	ConnStr string `json:"connstr"`
	Pwd     string `json:"pwd"`
}

func Rnd(min, max int) int {
	return min + rand.IntN(max-min)
}

func RndFrom(a ...string) string {
	return a[Rnd(0, len(a))]
}

func Dunno(g int) string {
	return RndFrom([2][]string{{"🤷‍♀️", "🤷🏻‍♀️", "🤷🏼‍♀️"}, {"🤷", "🤷‍♂️", "🤷🏻‍♂️"}}[g]...)
}

func Words(s, sep string) []string {
	if sep == "" {
		sep = " <>,./\\?!-=+*%^&$#`~(){}"
	}
	f := func(c rune) bool {
		return strings.Contains(sep, string(c))
	}
	return strings.FieldsFunc(s, f)
}

func ContainsWord(s, sub string) bool {
	if sub == "" {
		return false
	}
	sub = strings.ToLower(sub)
	for _, w := range Words(s, "") {
		w = strings.ToLower(w)
		switch {
		case w == "strike":
		case len(w) > 1 && strings.HasPrefix(w, sub):
			return true
		}
	}
	return false
}

func Min(v1, v2 int) int {
	if v1 <= v2 {
		return v1
	}
	return v2
}

func Max(v1, v2 int) int {
	if v1 >= v2 {
		return v1
	}
	return v2
}

func Left(src string, length int) string {
	r := []rune(src)
	if len(r) <= length {
		return src
	}
	return string(r[:length])
}

func TimeBetween_obsolete(s1, s2 string) bool {
	_check := func(s string) (int, error) {
		sl := strings.SplitN(s, ":", 3)
		if len(sl) == 1 {
			sl = append(sl, "0")
		}
		h, eH := strconv.Atoi(sl[0])
		m, eM := strconv.Atoi(sl[1])
		if eH != nil || eM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, errors.New("")
		}
		return h*60 + m, nil
	}
	i1, e1 := _check(s1)
	i2, e2 := _check(s2)
	if e1 != nil || e2 != nil {
		return false
	}
	i := time.Now().Hour()*60 + time.Now().Minute()
	return i >= i1 && i <= i2
}

func FileSize(filename string) int64 {
	info, e := os.Stat(filename)
	if os.IsNotExist(e) {
		return 0
	}
	return info.Size()
}

func FileExists(filename string) bool {
	info, e := os.Stat(filename)
	if os.IsNotExist(e) {
		return false
	}
	return !info.IsDir()
}

func Str2File(str, name string) error {
	if e := os.MkdirAll(path.Dir(name), 0755); e != nil {
		return e
	}
	f, e := os.Create(name)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.WriteString(str)
	return e
}

func R2file(r io.Reader, name string, by3rd bool) (e error) {
	var f *os.File
	if by3rd {
		f, e = os.CreateTemp("", "")
	} else {
		if e = os.MkdirAll(path.Dir(name), 0755); e == nil {
			f, e = os.Create(name)
		}
	}
	if e != nil {
		return
	}
	defer func() {
		if by3rd {
			if e == nil {
				f.Seek(0, 0)
				e = R2file(f, name, false)
			}
			f.Close()
			os.Remove(f.Name())
		} else {
			f.Close()
		}
	}()
	_, e = io.Copy(f, r)
	return
}

func CopyFile(src, dst string) error {
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()

	os.MkdirAll(path.Dir(dst), 0755)
	out, e := os.Create(dst)
	if e != nil {
		return e
	}
	defer out.Close()

	if _, e = io.Copy(out, in); e != nil {
		return e
	}
	return out.Sync()
}

func Iif[T any](b bool, v1, v2 T) T {
	if b {
		return v1
	}
	return v2
}

func UUID() (string, error) {
	uid, e := uuid.NewRandom()
	if e != nil {
		return "", e
	}
	return uid.String(), nil
}

func DeleteOldFiles(dir, mask string, days uint) error {
	if mask == "" {
		mask = ".*"
	}
	re, e := regexp.Compile(mask)
	if e != nil {
		return e
	}
	if _, e = os.Stat(dir); os.IsNotExist(e) {
		return e
	}
	de, e := os.ReadDir(dir)
	if e != nil {
		return e
	}

	for _, ent := range de {
		if !ent.IsDir() && re.MatchString(ent.Name()) {
			if info, e := ent.Info(); e == nil && time.Since(info.ModTime()).Hours() >= float64(time.Hour*24*time.Duration(days)) {
				os.Remove(filepath.Join(dir, ent.Name()))
			}
		}
	}
	return nil
}

func To1251(src string) string {
	if len(src) == 0 {
		return src
	}

	enc := charmap.Windows1251.NewEncoder()
	if s, _ := enc.String(src); len(s) == 0 {
		for _, r := range src {
			if s2, _ := enc.String(string(r)); len(s2) == 0 {
				continue
			}
			s += string(r)
		}
		src = s
	}
	return src
}

func ClearTime(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func NextTime(s string) time.Duration {
	t, _ := time.Parse("15:04", s)
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if now.After(target) {
		target = target.Add(24 * time.Hour)
	}
	return target.Sub(now)
}

func NowBetween(time1, time2 string) bool {
	t1, e1 := time.Parse("15:04", time1)
	t2, e2 := time.Parse("15:04", time2)
	tNow := time.Date(0, 1, 1, time.Now().Hour(), time.Now().Minute(), time.Now().Second(), 0, time.UTC)
	return errors.Join(e1, e2) == nil && tNow.After(t1) && tNow.Before(t2)
}
