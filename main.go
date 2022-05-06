package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type raftSession struct {
	Id     int `json:"id"`
	Leader struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	IsConnected     map[int]raftStatus `json:"isConnected"`
	AbleToReplicate raftStatus         `json:"ableToReplicate"`
	Error           string             `json:"error"`
}

type raftStatus bool

func (b *raftStatus) boolToFloat() float64 {
	if *b {
		return 1
	}
	return 0
}

func getScalityLiveCheck(url string) ([]raftSession, error) {
	c := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request")
	}
	req.Header.Set("User-Agent", "Go-Scality-Exporter")

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not perform HTTP request to URL: %v. Response: %v", url, res.Status)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not fetch status from %v: %v", url, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body")
	}

	var rss []raftSession
	jsonErr := json.Unmarshal(body, &rss)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to parse JSON response")
	}

	return rss, nil
}

func updateLivecheck(url string) {
	go func() {
		for {
			raftSessions, err := getScalityLiveCheck(url)

			if err != nil {
				fmt.Println("Error:", err)
			}

			for i := range raftSessions {
				for k, v := range raftSessions[i].IsConnected {
					rc.WithLabelValues(
						strconv.Itoa(raftSessions[i].Id),
						raftSessions[i].Leader.Host,
						strconv.Itoa(raftSessions[i].Leader.Port),
						strconv.Itoa(k),
						fmt.Sprintf(
							"%v:%v/%v",
							raftSessions[i].Leader.Host,
							strconv.Itoa(raftSessions[i].Leader.Port),
							strconv.Itoa(k),
						),
					).Add(v.boolToFloat())
				}

				rs.WithLabelValues(
					strconv.Itoa(raftSessions[i].Id),
					raftSessions[i].Leader.Host,
					strconv.Itoa(raftSessions[i].Leader.Port),
				).Add(raftSessions[i].AbleToReplicate.boolToFloat())
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

var (
	rs = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scality",
		Subsystem: "metadata_replication",
		Name:      "replication_status",
		Help:      "Status for Scality raft replication status",
	}, []string{
		// the raft session ID
		"id",
		// the IP address of the session leader
		"leader",
		// the port of the session leader
		"port",
	})
)

var (
	rc = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scality",
		Subsystem: "metadata_replication",
		Name:      "peer_connection",
		Help:      "Status for Scality raft session peering",
	}, []string{
		// the raft session ID
		"id",
		// the IP address of the session leader
		"leader",
		// the port of the session leader
		"port",
		// the ID of its metadata peer
		"connection_to",
		// label for unique grouping
		"connection_path",
	})
)

func main() {
	server := flag.String("server", "10.10.63.47", "IP address or FQDN")
	port := flag.String("port", "9000", "Port of `repd`")
	path := flag.String("path", "/_/livecheck", "Path to `livecheck`")

	flag.Parse()

	url := fmt.Sprintf("http://%v:%v%v", *server, *port, *path)

	updateLivecheck(url)

	fmt.Println("Exporter for Scality")
	fmt.Println("Listening on: http://0.0.0.0:9284/metrics")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9284", nil)
}
