package phones

import (
	"crypto/tls"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jesc7/zombot/cmd/zspy/client/types"
)

var pbUpdating bool

// PbUpdate - обновление открытой базы диапазонов номеров операторов
func PbUpdate(path string, files []string) (e error) {
	if pbUpdating {
		return nil
	}
	pbUpdating = true
	defer func() { pbUpdating = false }()

	var (
		pbFiles = [][]string{
			{"3", "ABC-3xx.csv"},
			{"4", "ABC-4xx.csv"},
			{"8", "ABC-8xx.csv"},
			{"9", "DEF-9xx.csv"},
		}
		pbURL = "https://opendata.digital.gov.ru/downloads/"
		upd   [][]string
	)
	if len(files) == 0 {
		upd = pbFiles
	} else {
		for _, v := range files {
			for _, v2 := range pbFiles {
				if v == v2[0] {
					upd = append(upd, v2)
					break
				}
			}
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 240 * time.Second,
	}
	wg := &sync.WaitGroup{}
	for _, v := range upd {
		wg.Add(1)
		go func(v []string) (e error) {
			defer func() {
				if e != nil {
					log.Printf("%s error: %v\n", v[1], e)
				}
				wg.Done()
			}()
			var req *http.Request
			if req, e = http.NewRequest(http.MethodGet, pbURL+v[1], nil); e != nil {
				return
			}

			req.Header.Set("User-Agent", "")
			resp, e := client.Do(req)
			if e != nil {
				return
			}

			defer resp.Body.Close()
			return types.R2file(resp.Body, filepath.Join(path, "phones", v[0]), true)
		}(v)
	}
	wg.Wait()
	return nil
}

func FindByPhone(path string, phone string) (res string) {
	phone = strings.NewReplacer("(", "", ")", "", "-", "", " ", "", "/", "", "\\", "").Replace(phone)
	if len(phone) < 10 {
		return
	}
	phone, file := phone[len(phone)-10:], ""
	switch first := phone[:1]; first {
	case "6", "7":
		return "Казахстан"

	default:
		file = filepath.Join(path, "phones", first)
		if types.FileSize(file) < 100 {
			PbUpdate(path, []string{first})
			if !types.FileExists(file) {
				return ""
			}
		}
	}

	p3, e := strconv.Atoi(phone[:3])
	if e != nil {
		return
	}
	p7, e := strconv.Atoi(phone[3:])
	if e != nil {
		return
	}
	f, e := os.Open(file)
	if e != nil {
		return
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 0
	r.LazyQuotes = true
	r.Comma = ';'
	r.Read()
	if r.FieldsPerRecord < 6 {
		return
	}
	for {
		rec, e := r.Read()
		if e == io.EOF {
			break
		}
		r0, e := strconv.Atoi(rec[0])
		if e != nil {
			return
		}
		if r0 == p3 {
			r1, e := strconv.Atoi(rec[1])
			if e != nil {
				return
			}
			r2, e := strconv.Atoi(rec[2])
			if e != nil {
				return
			}
			if p7 >= r1 && p7 <= r2 {
				return rec[5]
			}
		}
	}
	return
}
