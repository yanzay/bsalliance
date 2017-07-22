package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/yanzay/log"
	"github.com/yanzay/tbot"
)

const chatId = -1001119105956

type Player struct {
	Alliance string
	Name     string
}

type Immune struct {
	Player *Player
	End    time.Time
}

var (
	dbFile    = flag.String("data", "bsalliance.db", "Database file")
	adminUser = flag.String("admin", "yanzay", "Admin user")
)

var gameStore *GameStore

var immuneStandardDuration = 1 * time.Hour
var immuneConquerorDuration = 30 * time.Minute

var bot *tbot.Server

func main() {
	flag.Parse()
	gameStore = NewGameStore(*dbFile)
	gameStore.runWaiters()
	var err error
	bot, err = tbot.NewServer(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	bot.HandleFunc("/immunes", onlyUsers(immunesHandler))
	bot.HandleFunc("/delete {name}", onlyUsers(deleteHandler))
	bot.HandleFunc("/adduser {user}", onlyAdmin(addUserHandler))
	bot.HandleFunc("/deluser {user}", onlyAdmin(delUserHandler))
	bot.HandleFunc("/users", onlyAdmin(usersHandler))
	bot.HandleDefault(onlyUsers(parseForwardHandler))
	bot.ListenAndServe()
}

func addUserHandler(m *tbot.Message) {
	user := m.Vars["user"]
	if user == "" {
		m.Reply("User name required")
		return
	}
	gameStore.AddUser(user)
	m.Reply("OK")
}

func delUserHandler(m *tbot.Message) {
	user := m.Vars["user"]
	if user == "" {
		m.Reply("User name required")
		return
	}
	gameStore.DelUser(user)
	m.Reply("OK")
}

func usersHandler(m *tbot.Message) {
	users := gameStore.GetUsers()
	sendMarkdown(m, strings.Join(users, "\n"))
}

func immunesHandler(m *tbot.Message) {
	immunes := gameStore.GetImmunes()
	ims := make([]*Immune, 0)
	for _, immune := range immunes {
		ims = append(ims, immune)
	}
	sort.Slice(ims, func(i, j int) bool {
		return ims[i].End.After(ims[j].End)
	})
	lines := make([]string, 0)
	for _, immune := range ims {
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

func deleteHandler(m *tbot.Message) {
	gameStore.DeleteImmune(m.Vars["name"])
	m.Reply("Имун удален")
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
			immune, updated := gameStore.AddImmune(conqueror, forwardTime)
			if updated {
				go waiter(immune, fmt.Sprintf("Имун завоевателя закончился: %s", conqueror.Name))
			}
		}
		for _, player := range players {
			immune, updated := gameStore.AddImmune(player, forwardTime)
			if updated {
				go waiter(immune, fmt.Sprintf("Имун закончился: %s", player.Name))
			}
		}
		m.Replyf("%s: %s", printPlayers(players), forwardTime.String())
		quote, ok := maybeQuote()
		if ok {
			m.Reply(quote)
		}
	} else if strings.HasPrefix(m.Data, "‼️Битва с") {
		player := parseBattle(m.Data)
		if player != nil {
			immune, updated := gameStore.AddImmune(player, forwardTime)
			if updated {
				go waiter(immune, fmt.Sprintf("Имун закончился: %s", player.Name))
			}
			m.Replyf("%s: %s", player.Name, forwardTime.String())
			quote, ok := maybeQuote()
			if ok {
				m.Reply(quote)
			}
		}
	}
}

func waiter(immune *Immune, text string) {
	<-time.After(time.Until(immune.End))
	bot.Send(chatId, text)
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
