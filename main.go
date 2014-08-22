package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/oschwald/maxminddb-golang"
)

var (
	addr      = flag.String("addr", ":2342", "http service address")
	logAddr   = flag.String("logAddr", ":4223", "log line service address")
	geodbfile = flag.String("geodb", "GeoLite2-City.mmdb", "GeoIP2 / GeoLite2 db file")

	parserRex = flag.String("rex", "^.*nginxaccess: ([^ ]*) .*$", "regexp to match ip")
	requests  chan *request

	geodb  *maxminddb.Reader
	parser *regexp.Regexp

	errNoIPFound = errors.New("No IP found in log line")
	errUnknown   = errors.New("Location unknown")
)

type request struct {
	Lat  float64
	Lng  float64
	Name string
}

func serveHaProxy(ws *websocket.Conn) {
	for {
		req := <-requests
		log.Printf("Got neq request: %#v", req)
		data, err := json.Marshal(req)
		if err != nil {
			log.Print(err)
			return
		}
		ws.Write(data) // []byte(fmt.Sprintf("%f,%f", req.Lat, req.Long)))
	}
}

type onlyLocation struct {
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		MetroCode uint    `maxminddb:"metro_code"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}

func parseLine(line string) (*request, error) {
	matches := parser.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, errNoIPFound
	}
	ip := net.ParseIP(matches[1])
	if ip == nil {
		return nil, fmt.Errorf("Unexpected IP %s", matches[1])
	}
	var result onlyLocation
	if err := geodb.Lookup(ip, &result); err != nil {
		return nil, err
	}

	if result.Location.Latitude == 0 || result.Location.Longitude == 0 {
		return nil, errUnknown
	}
	req := &request{
		Lat: result.Location.Latitude,
		Lng: result.Location.Longitude,
	}
	return req, nil
}

func readLogs() {
	ln, err := net.Listen("tcp", *logAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if err := scanner.Err(); err != nil {
				log.Print(err)
				continue
			}

			req, err := parseLine(line)
			if err != nil {
				log.Print(err)
				continue
			}
			requests <- req
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	var err error
	parser, err = regexp.Compile(*parserRex)
	if err != nil {
		log.Fatal(err)
	}
	geodb, err = maxminddb.Open(*geodbfile)
	if err != nil {
		log.Fatal(err)
	}
	requests = make(chan *request)
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, index)
	})
	http.Handle("/haproxy", websocket.Handler(serveHaProxy))
	go readLogs()
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}
