package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

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
	Player *Player
	End    time.Time
}

var dbFile = flag.String("data", "bsalliance.db", "Database file")

var battleRegExp = regexp.MustCompile(`Битва с (\[[^[:ascii:]]*\])?(.*) окончена`)
var statRegExp = regexp.MustCompile(`Завоеватель:\W+(\w.*)`)

var gameStore *GameStore

var immuneStandardDuration = 1 * time.Hour
var immuneConquerorDuration = 30 * time.Minute

var bot *tbot.Server

func main() {
	flag.Parse()
	gameStore = NewGameStore(*dbFile)
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
		line := pad(immune.Player.Name, roundDuration(time.Until(immune.End)).String())
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
				<-time.After(time.Until(immune.End))
				bot.Send(chatId, fmt.Sprintf("Имун завоевателя закончился: %s", conqueror.Name))
			}()
		}
		var immune *Immune
		for _, player := range players {
			immune = gameStore.AddImmune(player, forwardTime)
		}
		go func() {
			<-time.After(time.Until(immune.End))
			bot.Send(chatId, fmt.Sprintf("Имун закончился: %s", printPlayers(players)))
		}()
		m.Replyf("%s: %s", printPlayers(players), forwardTime.String())
	} else if strings.HasPrefix(m.Data, "‼️Битва с") {
		player := parseBattle(m.Data)
		if player != nil {
			immune := gameStore.AddImmune(player, forwardTime)
			go func() {
				<-time.After(time.Until(immune.End))
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
