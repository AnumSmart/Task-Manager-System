package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BotHTTPClient представляет HTTP клиент для Telegram Bot API
// Инкапсулирует все необходимое для взаимодействия с Telegram
type BotHTTPClient struct {
	token   string       // Токен бота (получается от @BotFather)
	Http    *http.Client // HTTP клиент с настроенными таймаутами
	baseURL string       // Базовый URL для API запросов
}

// NewClient создает нового Telegram клиента
// token: токен бота в формате "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
// Возвращает готовый к работе клиент
func NewClient(token string) *BotHTTPClient {
	return &BotHTTPClient{
		token:   token,
		Http:    &http.Client{Timeout: 10 * time.Second},              // Важно: таймаут защищает от зависания запросов
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", token), // Формируем базовый URL согласно документации Telegram
	}
}

// SendMessage отправляет текстовое сообщение в чат
// chatID: ID получателя (пользователя или группы)
// text: текст сообщения
// replyMarkup: опциональная клавиатура (inline или обычная)
func (c *BotHTTPClient) SendMessage(chatID int64, text string, replyMarkup interface{}) error {
	// Формируем URL для метода sendMessage
	url := fmt.Sprintf("%s/sendMessage", c.baseURL)

	// Создаем тело запроса согласно документации Telegram API
	body := map[string]interface{}{
		"chat_id": chatID, // ID чата (обязательно)
		"text":    text,   // Текст сообщения (обязательно)
	}

	// Добавляем клавиатуру, если она предоставлена
	if replyMarkup != nil {
		body["reply_markup"] = replyMarkup
	}

	// Сериализуем в JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	// Отпрявляем запрос
	resp, err := c.Http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Простая структура для проверки успешности
	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Ok {
		return fmt.Errorf("failed to send message")
	}

	return nil
}
