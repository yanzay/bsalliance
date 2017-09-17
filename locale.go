package main

import "regexp"

func setEngLocale() {
	// Assets
	MessageNoImmunes = "There is no known immunes"
	MessageImmuneDeleted = "Immune deleted"
	MessageTrackImmune = "Track immune?"
	MessageTimeToFarm = "It's time to farm, lazy ass!"
	MessageEndOfImmune = "Immune ended: %s"
	MessageDontTrack = "OK, we will not"
	MessageConqueror = "Conqueror: %s"

	// Message parts
	ServerStatistics = "Server statistic"
	BattleWithAlliance = "‼️The battle with alliance"
	BattleWith = "‼️The battle with"

	// Buttons
	YesButton = "✅ Yes"
	NoButton = "❌ No"

	battleRegExp = regexp.MustCompile(`The battle with 🗡?[^[:ascii:]]?\[?([^[:ascii:]]*)?\]?(.*) complete`)
	statRegExp = regexp.MustCompile(`Conqueror:\s+(\[[^[:ascii:]]*\])?(.*)`)

	// Parser message parts
	Congratulations = "Congratulations"
	LosersPrefix = "Losers: "
	WinnersPrefix = "Winners: "
	LoseBattle = "Unfortunately"
	WinBattle = "Congratulations"

	quotes = []string{
		"Who controls the past controls the future. Who controls the present controls the past.",
		"War is peace. Freedom is slavery. Ignorance is strength.",
		"If you want to keep a secret, you must also hide it from yourself.",
		"We shall meet in the place where there is no darkness.",
		"In the face of pain there are no heroes.",
		"Big Brother is Watching You.",
		"Reality exists in the human mind, and nowhere else.",
		"We do not merely destroy our enemies; we change them.",
		"Sanity is not statistical.",
	}
}
