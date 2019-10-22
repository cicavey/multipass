package main

import (
	"bufio"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func loadConfig(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	log.Infoln("Loaded", len(lines), "DNs from", path)

	return lines, scanner.Err()
}

func main() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	log.Infoln("PID:", os.Getpid())

	// Hold config for atomic reloads
	var config atomic.Value

	confPath := flag.String("conf", "/etc/multipass/multipass.conf", "Multipass config file")
	addr := flag.String("addr", ":4444", "Listen address")
	hdr := flag.String("hdr", "X-Dn", "HTTP header containing DN")

	flag.Parse()

	// Load initial config and store it
	rawConfig, err := loadConfig(*confPath)
	if err != nil {
		log.Fatal(err)
	}
	config.Store(rawConfig)

	// Reload config on HUP
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP:
				rawConfig, err := loadConfig(*confPath)
				if err != nil {
					log.Fatal(err)
				}
				config.Store(rawConfig)
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				os.Exit(1)
			}
		}
	}()

	// Check all requests against valid list of DNs (assuming they already passed CA validation)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Check if a DN was at least passed, bail in no DN
		clientDNList, ok := r.Header[*hdr]
		if !ok || clientDNList == nil || len(clientDNList) == 0 {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		clientDN := strings.TrimSpace(clientDNList[0])

		validDNList := config.Load().([]string)

		isValid := false
		for _, testDN := range validDNList {
			if clientDN == testDN {
				isValid = true
				break
			}
		}

		log.Infoln(r.Header[*hdr], isValid)

		if isValid {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
	})

	log.Info("Listening on ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
