package server

import (
	"context"
	"log"

	tele "gopkg.in/telebot.v4"
)

// registerHandlers - регистрация обработчиков бота.
func (m *PollingBotManager) registerHandlers() {
	// 1. Команды
	m.telegramBot.Handle("/start", func(c tele.Context) error {
		// Отправляем задачу в канал вместо прямого вызова
		select {
		case m.workChan <- c:
			return nil
		case <-m.botCtx.Done():
			return context.Canceled
		default:
			// Канал переполнен
			c.Send("Извините, сервер перегружен. Попробуйте позже.")
			return nil
		}
	})

	m.telegramBot.Handle("/help", func(c tele.Context) error {
		select {
		case m.workChan <- c:
			return nil
		case <-m.botCtx.Done():
			return context.Canceled
		default:
			c.Send("Извините, сервер перегружен. Попробуйте позже.")
			return nil
		}
	})

	// 2. Callback
	m.telegramBot.Handle(tele.OnCallback, func(c tele.Context) error {
		select {
		case m.workChan <- c:
			return nil
		case <-m.botCtx.Done():
			return context.Canceled
		default:
			c.Respond(&tele.CallbackResponse{
				Text: "Извините, сервер перегружен, попробуйте позже",
			})
			return nil
		}
	})

	// 3. Текстовые сообщения
	m.telegramBot.Handle(tele.OnText, func(c tele.Context) error {
		select {
		case m.workChan <- c:
			return nil
		case <-m.botCtx.Done():
			return context.Canceled
		default:
			c.Send("Извините, сервер перегружен. Попробуйте позже.")
			return nil
		}
	})

	log.Println("📝 Обработчики бота зарегистрированы")
}
