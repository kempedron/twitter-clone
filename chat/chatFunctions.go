package functionsChat

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

var db *sql.DB

type Message struct {
	ID        int       `json:"ID"`
	ChatID    int       `json:"chat_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"send_time"`
}

// Инициализация базы данных
func InitDB() error {
	var err error
	db, err = sql.Open("postgres", "user=postgres password=322 dbname=twitter sslmode=disable")
	if err != nil {
		return err
	}
	return db.Ping() // Проверка соединения с базой данных
}

func GetMessageByrID(chatID int) ([]Message, error) {
	if db == nil {
		return nil, errors.New("database connection is not initialized")
	}

	query := `SELECT * FROM messages WHERE chat_id=$1 ORDER BY created_at`
	rows, err := db.Query(query, chatID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.ChatID, &message.UserID, &message.Content, &message.CreatedAt); err != nil {
			log.Println(err)
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func WriteMessageByID(msg Message) error {
	if db == nil {
		return errors.New("database connection is not initialized")
	}

	query := `INSERT INTO messages(chat_id, user_id, content) VALUES ($1, $2, $3) RETURNING id`
	if err := db.QueryRow(query, msg.ChatID, msg.UserID, msg.Content).Scan(&msg.ID); err != nil {
		return err
	}
	return nil
}

func GetMessages(c echo.Context) error {
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.Atoi(chatIDStr) // Преобразование строки в int
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный формат chat_id")
	}

	messages, err := GetMessageByrID(chatID)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Ошибка при получении сообщений")
	}
	return c.JSON(http.StatusOK, messages)
}

func PostMessage(c echo.Context) error {
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.Atoi(chatIDStr) // Преобразование строки в int
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный формат chat_id")
	}

	userID := c.Param("user_id")
	var msg Message
	if err := c.Bind(&msg); err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Ошибка при получении данных")
	}
	msg.ChatID = chatID // Присваиваем целое число
	msg.UserID = userID

	if err := WriteMessageByID(msg); err != nil {
		return c.String(http.StatusInternalServerError, "Ошибка при записи сообщения")
	}
	return c.JSON(http.StatusOK, msg)
}
