package main

import (
	"github.com/yanzay/tbot"
	"github.com/yanzay/tbot/model"
)

func onlyUsers(f tbot.HandlerFunction) tbot.HandlerFunction {
	return func(m *tbot.Message) {
		if gameStore.IsUser(m.From.UserName) || m.From.UserName == *adminUser || m.From.UserName == *cardinal {
			f(m)
			return
		}
		if m.ChatType != model.ChatTypePrivate {
			return
		}
		m.Reply("Access denied")
	}
}

func onlyAdmin(f tbot.HandlerFunction) tbot.HandlerFunction {
	return func(m *tbot.Message) {
		if m.From.UserName == *adminUser {
			f(m)
			return
		}
		m.Reply("Access denied")
	}
}
