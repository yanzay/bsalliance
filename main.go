package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yanzay/log"
	"github.com/yanzay/tbot"
)

const padWidth = 23
const chatId = -1001119105956

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
	immunes   map[string]*Immune
	conqueror *Player
}

var battleRegExp = regexp.MustCompile(`Битва с (\[[^[:ascii:]]*\])?(.*) окончена`)
var statRegExp = regexp.MustCompile(`Завоеватель:\W+(\w.*)`)

var gameStore = &GameStore{immunes: make(map[string]*Immune)}

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
		line := pad(immune.player.Name, roundDuration(time.Until(immune.end)).String())
		lines = append(lines, line)
	}
	reply := strings.Join(lines, "\n")
	if reply == "" {
		m.Reply("Известных иммунов нет")
		return
	}
	sendMarkdown(m, reply)
}

func sendMarkdown(m *tbot.Message, str string) {
	str = "```\n" + str + "```"
	m.Reply(str, tbot.WithMarkdown)
}

func parseForwardHandler(m *tbot.Message) {
	log.Println(m.ChatID)
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
	log.Println(m.Data)
	if strings.HasPrefix(m.Data, "‼️Битва с альянсом") {
		players := parseAllianceBattle(m.Data)
		if players == nil {
			return
		}
		conqueror, players := extractConqueror(players)
		if conqueror != nil {
			immune := gameStore.AddImmune(conqueror, forwardTime)
			go func() {
				<-time.After(time.Until(immune.end))
				bot.Send(chatId, fmt.Sprintf("Имун завоевателя закончился: %s", conqueror.Name))
			}()
		}
		var immune *Immune
		for _, player := range players {
			immune = gameStore.AddImmune(player, forwardTime)
		}
		go func() {
			<-time.After(time.Until(immune.end))
			bot.Send(chatId, fmt.Sprintf("Имун закончился: %s", printPlayers(players)))
		}()
		m.Replyf("%s: %s", printPlayers(players), forwardTime.String())
	} else if strings.HasPrefix(m.Data, "‼️Битва с") {
		player := parseBattle(m.Data)
		if player != nil {
			immune := gameStore.AddImmune(player, forwardTime)
			go func() {
				<-time.After(time.Until(immune.end))
				bot.Send(chatId, fmt.Sprintf("Имун закончился: %s", player.Name))
			}()
			m.Replyf("%s: %s", player.Name, forwardTime.String())
		}
	}
}

func printPlayers(players []*Player) string {
	names := make([]string, 0)
	for _, player := range players {
		names = append(names, player.Name)
	}
	return strings.Join(names, ", ")
}

func extractConqueror(players []*Player) (*Player, []*Player) {
	conqueror := gameStore.GetConqueror()
	for i, player := range players {
		if conqueror != nil && player.Name == conqueror.Name {
			return conqueror, append(players[:i], players[i+1:]...)
		}
	}
	return nil, players
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
	gs.immunes[player.Name] = immune
	return immune
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

func (gs *GameStore) GetImmunes() map[string]*Immune {
	gs.Lock()
	defer gs.Unlock()
	return gs.immunes
}

func roundDuration(d time.Duration) time.Duration {
	return d - (d % time.Second)
}

func pad(first, last string) string {
	fmt.Println(first, last)
	if utf8.RuneCountInString(first) > 16 {
		r := []rune(first)
		r = r[:16]
		first = string(r)
	}
	repeatCount := padWidth - utf8.RuneCountInString(first) - utf8.RuneCountInString(last)
	if repeatCount <= 0 {
		repeatCount = 1
	}
	return first + strings.Repeat(".", repeatCount) + last
}
