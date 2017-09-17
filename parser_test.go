package main

import "testing"

var statMessageWithAlliance = `💾Статистика сервера

Пользователи    
🔅Всего             15925
🔅Зарегистрировано   9092
🔅С казармами        2157
🔅Активных за день    375

🗡Завоеватель:    [😈]Батон

🏁Дней с запуска      196`

var statMessageWithoutAlliance = `💾Статистика сервера

Пользователи    
🔅Всего             15925
🔅Зарегистрировано   9092
🔅С казармами        2157
🔅Активных за день    375

🗡Завоеватель:    Батон

🏁Дней с запуска      196`

var (
	battleMessageWithConqueror                = `‼️Битва с 🗡[😈]Батон окончена. Поздравляю, Ильгиз! Твоя армия одержала победу. Победители 11344⚔ из 13320⚔ гордо возвращаются домой. Твоя награда составила 1038648💰, a 28384🗺 отошли к твоим владениям. Твоя карма изменилась на 3☯.`
	battleMessageWithoutConqueror             = `‼️Битва с [🐉]Василий Великий окончена. Поздравляю, Dimonstr! Твоя армия одержала победу. Победители 12080⚔ без единой потери гордо возвращаются домой. Твоя награда составила 20💰, a 2263🗺 отошли к твоим владениям. Твоя карма изменилась на 3☯.`
	battleMessageWithConquerorWithoutAlliance = ` ‼Битва с 🗡Cuclas окончена. Поздравляю, Darksoul! Твой альянс одержал победу. Победители 4883⚔ из 10000⚔ гордо возвращаются домой. Твоя награда составила 307046💰, a 9720🗺 отошли к твоим владениям. Твоя карма изменилась на 2☯.`
	notConqueror                              = "Василий Великий"
	notConquerorAlliance                      = "🐉"
	conqueror                                 = "Батон"
	conquerorAlliance                         = "😈"
	conquerorWithoutAlliance                  = "Cuclas"
)

func TestParseConqueror(t *testing.T) {
	player := parseConqueror(statMessageWithAlliance)
	if player.Name != conqueror {
		t.Errorf("expected conqueror: %s, actual: %s", conqueror, player.Name)
	}
	player = parseConqueror(statMessageWithoutAlliance)
	if player.Name != conqueror {
		t.Errorf("expected conqueror: %s, actual: %s", conqueror, player.Name)
	}
}

func TestParseBattle(t *testing.T) {
	player := parseBattle(battleMessageWithoutConqueror)
	if player.Name != notConqueror {
		t.Errorf("expected player name: %s, actual: %s", notConqueror, player.Name)
	}
	if player.Alliance != notConquerorAlliance {
		t.Errorf("expected player alliance: %s, actual: %s", notConquerorAlliance, player.Alliance)
	}
	player = parseBattle(battleMessageWithConqueror)
	if player.Name != conqueror {
		t.Errorf("expected player name: %s, actual: %s", conqueror, player.Name)
	}
	if player.Alliance != conquerorAlliance {
		t.Errorf("expected player alliance: %s, actual: %s", conquerorAlliance, player.Alliance)
	}
	player = parseBattle(battleMessageWithConquerorWithoutAlliance)
	if player.Name != conquerorWithoutAlliance {
		t.Errorf("expected player name: %s, actual: %s", conquerorWithoutAlliance, player.Name)
	}
	if player.Alliance != "" {
		t.Errorf("expected no alliance, actual: %s", player.Alliance)
	}
}
