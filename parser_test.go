package main

import "testing"

var statMessage = `💾Статистика сервера

Пользователи    
🔅Всего             15925
🔅Зарегистрировано   9092
🔅С казармами        2157
🔅Активных за день    375

🗡Завоеватель:    [😈]Батон

🏁Дней с запуска      196`

var conqueror = "Батон"

func TestParseConqueror(t *testing.T) {
	player := parseConqueror(statMessage)
	if player.Name != conqueror {
		t.Errorf("expected conqueror: %s, actual: %s", conqueror, player.Name)
	}
}
