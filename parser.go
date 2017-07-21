package main

import (
	"log"
	"regexp"
	"strings"
)

var (
	battleRegExp = regexp.MustCompile(`Битва с ([^[:ascii:]]?\[[^[:ascii:]]*\])?(.*) окончена`)
	statRegExp   = regexp.MustCompile(`Завоеватель:\s+(\[[^[:ascii:]]*\])?(.*)`)
)

func parseConqueror(message string) *Player {
	matches := statRegExp.FindStringSubmatch(message)
	if len(matches) < 3 {
		return nil
	}
	return &Player{Name: matches[2]}
}

func parseBattle(message string) *Player {
	if !battleAttack(message) {
		return nil
	}
	matches := battleRegExp.FindStringSubmatch(message)
	if len(matches) < 3 {
		return nil
	}
	log.Printf("Alliance: %s", matches[1])
	log.Printf("Name: %s", matches[2])
	return &Player{Alliance: matches[1], Name: matches[2]}
}

func parseAllianceBattle(message string) []*Player {
	if strings.Contains(message, "Поздравляю") {
		return parseWinAllianceBattle(message)
	}
	return parseLoseAllianceBattle(message)
}

func parseWinAllianceBattle(message string) []*Player {
	if strings.Contains(message, "🗺") {
		return parseLosers(message)
	}
	return nil
}

func parseLoseAllianceBattle(message string) []*Player {
	if !strings.Contains(message, "🗺") {
		return parseWinners(message)
	}
	return nil
}

func parseLosers(message string) []*Player {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Проигравшие: ") {
			loseStr := strings.TrimPrefix(line, "Проигравшие: ")
			players := make([]*Player, 0)
			names := strings.Split(loseStr, ", ")
			for _, name := range names {
				players = append(players, &Player{Name: name})
			}
			return players
		}
	}
	return nil
}

func parseWinners(message string) []*Player {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Победители: ") {
			loseStr := strings.TrimPrefix(line, "Победители: ")
			players := make([]*Player, 0)
			names := strings.Split(loseStr, ", ")
			for _, name := range names {
				players = append(players, &Player{Name: name})
			}
			return players
		}
	}
	return nil
}

func battleAttack(message string) bool {
	return strings.Contains(message, "К сожалению") && !strings.Contains(message, "🗺") ||
		strings.Contains(message, "Поздравляю") && strings.Contains(message, "🗺")
}
