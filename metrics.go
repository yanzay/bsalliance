package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/yanzay/log"
	"github.com/yanzay/tbot"
)

var (
	dbClient      client.Client
	allyMembersRx = regexp.MustCompile(`[🔅⚜️👑](.*)\s(\d+)🛡`)
)

func allianceForwardHandler(m *tbot.Message) {
	if m.ForwardDate == 0 {
		return
	}
	forwardTime := time.Unix(int64(m.ForwardDate), 0)
	levels := parseLevels(m.Text())
	save(forwardTime, levels)
	m.Reply("Thank you!")
}

func parseLevels(text string) map[string]int {
	levels := map[string]int{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		matches := allyMembersRx.FindStringSubmatch(line)
		name := strings.TrimSpace(matches[1])
		var level int
		fmt.Sscanf(matches[2], "%d", &level)
		levels[name] = level
	}
	return levels
}

func initDB() {
	var err error
	dbClient, err = client.NewHTTPClient(client.HTTPConfig{Addr: *influxDB})
	if err != nil {
		log.Fatal(err)
	}
}

func save(timestamp time.Time, levels map[string]int) {
	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  *influxDBName,
		Precision: "s",
	})
	if err != nil {
		log.Errorf("can't create batch points: %s", err)
		return
	}

	for name, level := range levels {
		// Create a point and add to batch
		tags := map[string]string{"name": name}
		fields := map[string]interface{}{"level": level}

		pt, err := client.NewPoint("barracks", tags, fields, timestamp)
		if err != nil {
			log.Errorf("can't create new point: %q", err)
			continue
		}
		bp.AddPoint(pt)
	}

	// Write the batch
	err = dbClient.Write(bp)
	if err != nil {
		log.Errorf("can't write batch to db: %q", err)
		return
	}
}

func isAllianceMembers(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}
	matches := allyMembersRx.FindStringSubmatch(lines[0])
	if len(matches) != 3 {
		return false
	}
	return true
}
