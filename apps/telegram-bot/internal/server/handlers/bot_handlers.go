package handlers

import (
	tele "gopkg.in/telebot.v4"
)

// BotHttpHandler - обработчик для Telegram бота.
type BotHttpHandler struct {
}

// NewBotHttpHandler - конструктор.
func NewBotHttpHandler() (*BotHttpHandler, error) {
	return &BotHttpHandler{}, nil
}

// Обработка всех текстовых сообщений.
func (b *BotHttpHandler) HandleBotMessage(c tele.Context) error {
	return nil
}

// Обработка callback-запросов от inline клавиатур.
func (b *BotHttpHandler) HandleBotCallback(c tele.Context) error {
	return nil
}

// Обработка команды /start.
func (b *BotHttpHandler) HandleBotStart(c tele.Context) error {
	return nil
}

// Обработка команды /help.
func (b *BotHttpHandler) HandleBotHelp(c tele.Context) error {
	return nil
}
