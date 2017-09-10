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
var (
	MessageNoImmunes     = "Известных иммунов нет"
	MessageImmuneDeleted = "Имун удален"
	MessageTrackImmune   = "Отслеживать иммун?"
	MessageTimeToFarm    = "Пора на ферму, ленивая задница!"
	MessageEndOfImmune   = "Имун закончился: %s"
	MessageDontTrack     = "Хорошо, не будем"
	MessageConqueror     = "Завоеватель: %s"
)

// Message parts
var (
	ServerStatistics   = "Статистика сервера"
	BattleWithAlliance = "‼️Битва с альянсом"
	BattleWith         = "‼️Битва с"

	ServerStatisticsRu   = "Статистика сервера"
	BattleWithAllianceRu = "‼️Битва с альянсом"
	BattleWithRu         = "‼️Битва с"
)

// Buttons
var (
	YesButton = "✅ Да"
	NoButton  = "❌ Нет"
)

var (
	dbFile    = flag.String("data", "bsalliance.db", "Database file")
	adminUser = flag.String("admin", "yanzay", "Admin user")
	chatID    = flag.Int64("chat", -1001119105956, "Chat ID for reporting")
	eng       = flag.Bool("eng", false, "English locale")
	cardinal  = flag.String("c", "", "Cardinal user")
	noClear   = flag.Bool("no-clear", false, "Disable clear command")
)

var gameStore *GameStore

var immuneStandardDuration = 1 * time.Hour
var immuneConquerorDuration = 30 * time.Minute

var bot *tbot.Server

func init() {
	flag.Parse()
	if *eng {
		setEngLocale()
	}
}

func logger(f tbot.HandlerFunction) tbot.HandlerFunction {
	return func(m *tbot.Message) {
		log.Infof("[%d] %s (%s): %s", m.ChatID, m.From.FirstName, m.From.UserName, m.Text())
		f(m)
	}
}

func main() {
	gameStore = NewGameStore(*dbFile)
	gameStore.runWaiters()
	var err error
	bot, err = tbot.NewServer(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	bot.AddMiddleware(logger)
	bot.HandleFunc("/immunes", onlyUsers(immunesHandler))
	bot.HandleFunc("/clear", onlyUsers(clearHandler))
	bot.HandleFunc("/delete {name}", onlyUsers(deleteHandler))
	bot.HandleFunc("/adduser {user}", onlyAdmin(addUserHandler))
	bot.HandleFunc("/deluser {user}", onlyAdmin(delUserHandler))
	bot.HandleFunc("/users", onlyUsers(usersHandler))
	bot.HandleDefault(onlyUsers(parseForwardHandler))
	bot.ListenAndServe()
}

func clearHandler(m *tbot.Message) {
	if *noClear {
		m.Reply("NO! NO! GOD, NO!")
		return
	}
	gameStore.ClearImmunes()
	m.Reply("OK")
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
	if m.ChatType != model.ChatTypePrivate {
		return
	}
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
	if m.ChatType != model.ChatTypePrivate {
		return
	}
	var replyTo = m.ChatID
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
	case m.ForwardDate == 0:
	case isServerStatistics(m.Data):
		conqueror := parseConqueror(m.Data)
		gameStore.SetConqueror(conqueror)
		m.Replyf(MessageConqueror, gameStore.GetConqueror().Name)
	case isBattleWithAlliance(m.Data):
		players := parseAllianceBattle(m.Data)
		if players == nil {
			return
		}
		for _, player := range players {
			updateImmune(player, forwardTime, replyTo)
		}
		m.Replyf("%s: %s", printPlayers(players), forwardTime.String())
	case isBattleWith(m.Data):
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

func isBattleWithAlliance(msg string) bool {
	return strings.HasPrefix(msg, BattleWithAlliance) ||
		strings.HasPrefix(msg, BattleWithAllianceRu)
}

func isBattleWith(msg string) bool {
	return strings.HasPrefix(msg, BattleWith) ||
		strings.HasPrefix(msg, BattleWithRu)
}

func isServerStatistics(msg string) bool {
	return strings.Contains(msg, ServerStatistics) ||
		strings.Contains(msg, ServerStatisticsRu)
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
	saved := gameStore.GetImmune(immune.Player.Name)
	if saved == nil {
		return
	}
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
