package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yanzay/log"
	"github.com/yanzay/tbot"
)

type Player struct {
	Alliance string
	Name     string
}

type Immune struct {
	player *Player
	end    time.Time
}

type GameStore struct {
	sync.Mutex
	immunes   []*Immune
	conqueror *Player
}

var battleRegExp = regexp.MustCompile(`Битва с\W+(.*) окончена.`)
var statRegExp = regexp.MustCompile(`Завоеватель:\W+(\w.*)`)

var gameStore = &GameStore{immunes: make([]*Immune, 0)}

var immuneStandardDuration = 1 * time.Hour
var immuneConquerorDuration = 30 * time.Minute

var bot *tbot.Server

func main() {
	var err error
	bot, err = tbot.NewServer(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	bot.HandleFunc("/immunes", immunesHandler)
	bot.HandleDefault(parseForwardHandler)
	bot.ListenAndServe()
}

func immunesHandler(m *tbot.Message) {
	lines := make([]string, 0)
	immunes := gameStore.GetImmunes()
	for _, immune := range immunes {
		line := fmt.Sprintf("%s: %s", immune.player.Name, immune.end.Sub(time.Now()))
		lines = append(lines, line)
	}
	reply := strings.Join(lines, "\n")
	if reply == "" {
		m.Reply("Известных иммунов нет")
		return
	}
	m.Reply(strings.Join(lines, "\n"))
}

func parseForwardHandler(m *tbot.Message) {
	if m.ForwardDate == 0 {
		return
	}
	if strings.Contains(m.Data, "Статистика сервера") {
		conqueror := parseConqueror(m.Data)
		gameStore.SetConqueror(conqueror)
		m.Replyf("Завоеватель: %s", gameStore.GetConqueror().Name)
		return
	}
	forwardTime := time.Unix(int64(m.ForwardDate), 0)
	log.Println(m)
	log.Println(m.Data)
	if strings.HasPrefix(m.Data, "‼️Битва с альянсом") {
		names := parseAllianceBattle(m.Data)
		if names == nil {
			return
		}
		m.Replyf("%s: %s", strings.Join(names, ", "), forwardTime.String())
	} else if strings.HasPrefix(m.Data, "‼️Битва с") {
		player := parseBattle(m.Data)
		if player != nil {
			immune := gameStore.AddImmune(player, forwardTime)
			go func() {
				<-time.After(immune.end.Sub(time.Now()))
				gameStore.RemoveImmune(player)
				bot.Send(m.ChatID, fmt.Sprintf("Имун закончился: %s", player.Name))
			}()
			m.Replyf("%s: %s", player.Name, forwardTime.String())
		}
	}
}

func parseAllianceBattle(message string) []string {
	if strings.Contains(message, "Поздравляю") {
		return parseWinAllianceBattle(message)
	}
	return parseLoseAllianceBattle(message)
}

func parseWinAllianceBattle(message string) []string {
	if strings.Contains(message, "🗺") {
		return parseLosers(message)
	}
	return nil
}

func parseLoseAllianceBattle(message string) []string {
	if !strings.Contains(message, "🗺") {
		return parseWinners(message)
	}
	return nil
}

func parseLosers(message string) []string {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Проигравшие: ") {
			loseStr := strings.TrimPrefix(line, "Проигравшие: ")
			return strings.Split(loseStr, ", ")
		}
	}
	return nil
}

func parseWinners(message string) []string {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Победители: ") {
			loseStr := strings.TrimPrefix(line, "Победители: ")
			return strings.Split(loseStr, ", ")
		}
	}
	return nil
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

func battleAttack(message string) bool {
	return strings.Contains(message, "К сожалению") && !strings.Contains(message, "🗺") ||
		strings.Contains(message, "Поздравляю") && strings.Contains(message, "🗺")
}

func (gs *GameStore) AddImmune(player *Player, start time.Time) *Immune {
	gs.Lock()
	defer gs.Unlock()
	var end time.Time
	if gs.conqueror != nil && gs.conqueror.Name == player.Name {
		end = start.Add(immuneConquerorDuration)
	} else {
		end = start.Add(immuneStandardDuration)
	}
	immune := &Immune{player: player, end: end}
	gs.immunes = append(gs.immunes, immune)
	return immune
}

func (gs *GameStore) RemoveImmune(player *Player) {
	gs.Lock()
	defer gs.Unlock()
	for i, immune := range gs.immunes {
		if immune.player.Name == player.Name {
			gs.immunes = append(gs.immunes[:i], gs.immunes[i+1:]...)
		}
	}
}

func (gs *GameStore) SetConqueror(player *Player) {
	gs.Lock()
	gs.conqueror = player
	gs.Unlock()
}

func (gs *GameStore) GetConqueror() *Player {
	gs.Lock()
	conqueror := gs.conqueror
	gs.Unlock()
	return conqueror
}

func parseConqueror(message string) *Player {
	matches := statRegExp.FindStringSubmatch(message)
	if len(matches) < 2 {
		return nil
	}
	return &Player{Name: matches[1]}
}

func (gs *GameStore) GetImmunes() []*Immune {
	gs.Lock()
	defer gs.Unlock()
	return gs.immunes
}
