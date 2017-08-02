package main

import (
	"regexp"
	"strings"
)

var (
	battleRegExp = regexp.MustCompile(`Битва с ([^[:ascii:]]?\[[^[:ascii:]]*\])?(.*) окончена`)
	statRegExp   = regexp.MustCompile(`Завоеватель:\s+(\[[^[:ascii:]]*\])?(.*)`)
)

// Message parts
var (
	Congratulations = "Поздравляю"
	LosersPrefix    = "Проигравшие: "
	WinnersPrefix   = "Победители: "
	LoseBattle      = "К сожалению"
	WinBattle       = "Поздравляю"
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
	return &Player{Alliance: matches[1], Name: matches[2]}
}

func parseAllianceBattle(message string) []*Player {
	if strings.Contains(message, Congratulations) {
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
		if strings.HasPrefix(line, LosersPrefix) {
			loseStr := strings.TrimPrefix(line, LosersPrefix)
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
		if strings.HasPrefix(line, WinnersPrefix) {
			winStr := strings.TrimPrefix(line, WinnersPrefix)
			players := make([]*Player, 0)
			names := strings.Split(winStr, ", ")
			for _, name := range names {
				players = append(players, &Player{Name: name})
			}
			return players
		}
	}
	return nil
}

func battleAttack(message string) bool {
	return strings.Contains(message, LoseBattle) && !strings.Contains(message, "🗺") ||
		strings.Contains(message, WinBattle) && strings.Contains(message, "🗺")
}
