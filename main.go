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
	"github.com/yanzay/tbot/model"
)

type Player struct {
	Alliance string
	Name     string
}

type Immune struct {
	Player *Player
	End    time.Time
}

// Assets
const (
	MessageNoImmunes     = "Известных иммунов нет"
	MessageImmuneDeleted = "Имун удален"
	MessageTrackImmune   = "Отслеживать иммун?"
	MessageTimeToFarm    = "Пора на ферму, ленивая задница!"
	MessageEndOfImmune   = "Имун закончился: %s"
	MessageDontTrack     = "Хорошо, не будем"
)

// Buttons
const (
	YesButton = "✅ Да"
	NoButton  = "❌ Нет"
)

var (
	dbFile    = flag.String("data", "bsalliance.db", "Database file")
	adminUser = flag.String("admin", "yanzay", "Admin user")
	chatID    = flag.Int64("chat", -1001119105956, "Chat ID for reporting")
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
		m.Reply(MessageNoImmunes)
		return
	}
	sendMarkdown(m, reply)
}

func deleteHandler(m *tbot.Message) {
	gameStore.DeleteImmune(m.Vars["name"])
	m.Reply(MessageImmuneDeleted)
}

func sendMarkdown(m *tbot.Message, str string) {
	str = "```\n" + str + "```"
	m.Reply(str, tbot.WithMarkdown)
}

func parseForwardHandler(m *tbot.Message) {
	log.Info(m.ChatID)
	log.Info(m.Data)
	var replyTo int64
	if m.ChatType == model.ChatTypePrivate {
		replyTo = m.ChatID
	}
	forwardTime := time.Unix(int64(m.ForwardDate), 0)
	switch {
	case m.Data == YesButton:
		if responses[m.ChatID] != nil {
			select {
			case responses[m.ChatID] <- true:
			default:
			}
		}
	case m.Data == NoButton:
		if responses[m.ChatID] != nil {
			select {
			case responses[m.ChatID] <- false:
			default:
			}
		}
	case strings.Contains(m.Data, "Статистика сервера"):
		conqueror := parseConqueror(m.Data)
		gameStore.SetConqueror(conqueror)
		m.Replyf("Завоеватель: %s", gameStore.GetConqueror().Name)
	case strings.HasPrefix(m.Data, "‼️Битва с альянсом"):
		players := parseAllianceBattle(m.Data)
		if players == nil {
			return
		}
		for _, player := range players {
			updateImmune(player, forwardTime, replyTo)
		}
		m.Replyf("%s: %s", printPlayers(players), forwardTime.String())
	case strings.HasPrefix(m.Data, "‼️Битва с"):
		player := parseBattle(m.Data)
		if player != nil {
			if replyTo != 0 {
				go farmer(forwardTime.Add(10*time.Minute), replyTo)
				add := askToAdd(m)
				if !add {
					m.Reply(MessageDontTrack)
					return
				}
			}
			updateImmune(player, forwardTime, replyTo)
			m.Replyf("%s: %s", player.Name, forwardTime.String())
		}
	}
	quote, ok := maybeQuote()
	if ok {
		m.Reply(quote)
	}
}

func updateImmune(player *Player, forwardTime time.Time, replyTo int64) {
	immune, updated := gameStore.AddImmune(player, forwardTime)
	if updated {
		go waiter(immune, fmt.Sprintf(MessageEndOfImmune, player.Name), replyTo)
	}
}

var responses = make(map[int64]chan bool)

func askToAdd(m *tbot.Message) bool {
	buttons := []string{YesButton, NoButton}
	m.ReplyKeyboard(MessageTrackImmune, [][]string{buttons}, tbot.OneTimeKeyboard)
	responses[m.ChatID] = make(chan bool)
	select {
	case answer := <-responses[m.ChatID]:
		return answer
	case <-time.After(1 * time.Minute):
		return false
	}
}

func farmer(end time.Time, replyTo int64) {
	<-time.After(time.Until(end))
	bot.Send(replyTo, MessageTimeToFarm)
}

func waiter(immune *Immune, text string, replyTo int64) {
	<-time.After(time.Until(immune.End))
	bot.Send(*chatID, text)
	if replyTo != 0 {
		bot.Send(replyTo, text)
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
