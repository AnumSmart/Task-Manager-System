package server

import (
	tele "gopkg.in/telebot.v4"
)

// registerHandlers - регистрация обработчиков
func (m *PollingBotManager) registerHandlers() {
	// Единый обработчик для всех команд
	submitTask := func(c tele.Context) error {
		return m.workerPool.Submit(&BotTask{
			ctx:     c,
			handler: m.handler,
		})
	}

	// Регистрируем все обработчики
	m.telegramBot.Handle("/start", submitTask)
	m.telegramBot.Handle("/help", submitTask)
	m.telegramBot.Handle(tele.OnText, submitTask)
	m.telegramBot.Handle(tele.OnCallback, submitTask)
}
