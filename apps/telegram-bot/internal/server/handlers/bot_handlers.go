package handlers

import (
	"context"
	"log"

	tele "gopkg.in/telebot.v4"
)

// BotHttpHandler - обработчик для Telegram бота.
type BotHttpHandler struct {
	bot *tele.Bot
}

// NewBotHttpHandler - конструктор.
func NewBotHttpHandler(bot *tele.Bot) (*BotHttpHandler, error) {
	return &BotHttpHandler{bot: bot}, nil
}

// ProcessStart - обработка команды /start
func (b *BotHttpHandler) ProcessStart(ctx context.Context, chatID int64, senderID int64) error {
	log.Printf("User %d started bot in chat %d", senderID, chatID)

	// Бизнес-логика
	// ...

	// Отправка ответа (используем сохраненный экземпляр бота)
	_, err := b.bot.Send(&tele.Chat{ID: chatID}, "Привет! Я бот для управления задачами.")
	return err
}

// ProcessHelp - обработка команды /help
func (b *BotHttpHandler) ProcessHelp(ctx context.Context, chatID int64, senderID int64) error {
	_, err := b.bot.Send(&tele.Chat{ID: chatID}, "Список доступных команд:\n/start - начать\n/help - помощь")
	return err
}

// ProcessCallback - обработка нажатия на кнопку
func (b *BotHttpHandler) ProcessCallback(ctx context.Context, chatID int64, senderID int64, callbackID, callbackData string, messageID int) error {
	log.Printf("Callback from user %d: %s", senderID, callbackData)

	// Обработка callback
	switch callbackData {
	case "btn_yes":
		b.bot.Send(&tele.Chat{ID: chatID}, "Вы нажали ДА")
	case "btn_no":
		b.bot.Send(&tele.Chat{ID: chatID}, "Вы нажали НЕТ")
	}

	// Обязательно отвечаем на callback, чтобы убрать "часики" у кнопки
	return b.bot.Respond(&tele.Callback{ID: callbackID}, &tele.CallbackResponse{
		Text: "Готово!",
	})
}

// ProcessMessage - обработка текстовых сообщений
func (b *BotHttpHandler) ProcessMessage(ctx context.Context, chatID int64, senderID int64, text string) error {
	log.Printf("Message from %d: %s", senderID, text)

	// Бизнес-логика
	// Например, парсинг текста как запроса
	response := b.parseUserRequest(text)

	_, err := b.bot.Send(&tele.Chat{ID: chatID}, response)
	return err
}

// ProcessUnknown - обработка неизвестного типа
func (b *BotHttpHandler) ProcessUnknown(ctx context.Context, chatID int64, senderID int64) error {
	_, err := b.bot.Send(&tele.Chat{ID: chatID}, "Я не понял это сообщение 😕")
	return err
}

// Вспомогательная функция
func (b *BotHttpHandler) parseUserRequest(text string) string {
	// Здесь логика парсинга
	return "Ваш запрос: " + text
}
