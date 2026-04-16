package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	max "github.com/max-messenger/max-bot-api-client-go"
)

type Config struct {
	Max struct {
		Token string `json:"token"`
	} `json:"max"`
	Proxy struct {
		Addr string `json:"addr"`
		Port int    `json:"port"`
	} `json:"proxy"`
}

func main() {
	f, e := os.ReadFile(filepath.Join(filepath.Dir(os.Args[0]), "cfg.json"))
	if e != nil {
		log.Fatalln("Can't read config file:", e)
	}
	var cfg Config
	if e = json.Unmarshal(f, &cfg); e != nil {
		log.Fatalln("Can't unmarshal the json:", e)
	}

	var options []max.Option
	if cfg.Proxy.Addr != "" {
		var proxy *url.URL
		if proxy, e = url.Parse(fmt.Sprintf("%s:%d", cfg.Proxy.Addr, cfg.Proxy.Port)); e == nil {
			options = append(options, max.WithHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxy),
					},
				},
			))
		}
	}
	bot, e := max.New(cfg.Max.Token, options...)
	if e != nil {
		log.Fatalln("Can't create bot:", e)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	info, e := bot.Bots.GetBot(ctx)
	fmt.Printf("Get me: %#v %#v", info, e)
}
