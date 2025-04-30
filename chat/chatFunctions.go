package functionsChat

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	db "twitter/DataBase"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

type Message struct {
	ID        int    `json:"ID"`
	ChatID    int    `json:"chat_id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	CreatedAt string `json:"send_time"`
}

type ChatPageData struct {
	ChatID   int
	UserID   string
	Messages []Message
}

func GetMessageByrID(chatID int) (ChatPageData, error) {
	db := db.Get()
	if db == nil {
		return ChatPageData{}, errors.New("database connection is not initialized")
	}
	var userID string
	var username string

	if err := db.QueryRow("SELECT user_id FROM messages WHERE chat_id=$1", chatID).Scan(&userID); err != nil {
		log.Println(err)
		return ChatPageData{}, err
	}

	if err := db.QueryRow("SELECT username FROM users WHERE id=$1", userID).Scan(&username); err != nil {
		log.Println(err)
		return ChatPageData{}, err
	}

	query := `SELECT id, chat_id, user_id, (SELECT username FROM users WHERE id=user_id) AS username, content, TO_CHAR(created_at, 'DD Mon YYYY HH24:MI:SS') AS formatted_time FROM messages WHERE chat_id=$1 ORDER BY created_at`
	rows, err := db.Query(query, chatID)
	if err != nil {
		log.Println(err)
		return ChatPageData{}, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.ChatID, &message.UserID, &message.Username, &message.Content, &message.CreatedAt); err != nil {
			log.Println(err)
			return ChatPageData{}, err
		}
		messages = append(messages, message)
	}
	data := ChatPageData{
		ChatID:   chatID,
		UserID:   userID,
		Messages: messages,
	}
	return data, nil
}

func WriteMessageByID(msg Message) error {
	db := db.Get()

	if db == nil {
		return errors.New("database connection is not initialized")
	}

	query := `INSERT INTO messages(chat_id, user_id, content) VALUES ($1, $2, $3) RETURNING id`
	if err := db.QueryRow(query, msg.ChatID, msg.UserID, msg.Content).Scan(&msg.ID); err != nil {
		log.Println("ошибка:", err)
		return err
	}
	return nil
}

func GetMessages(c echo.Context) error {
	chatIDStr := c.Param("chat_id")
	if chatIDStr == "" {
		log.Println("не должно быть пустым")
		return c.String(http.StatusInternalServerError, "не должно быть пустым")
	}
	chatID, err := strconv.Atoi(chatIDStr) // Преобразование строки в int
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный формат chat_id")
	}

	messages, err := GetMessageByrID(chatID)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Ошибка при получении сообщений")
	}
	return c.Render(http.StatusOK, "chatPage.html", messages)
}

// структура для чата
type Chat struct {
	ChatID    int    `json:"chat_id"`
	UserID1   int    `json:"user_id1"`
	UserID2   int    `json:"user_id2"`
	UserName1 string `json:"username1"`
	UserName2 string `json:"username2"`
}

func GetChatsByUsername(username string) ([]Chat, error) {
	db := db.Get()

	var chats []Chat
	query := `SELECT 
	chat_id,
	user_id_1,
	user_id_2,
	(SELECT username from users WHERE id=user_id_1) AS user1_name,
	(SELECT username from users WHERE id=user_id_2) AS user2_name

	 FROM chats 
	 WHERE user_id_1 = (select id from users WHERE username=$1) 
	 or user_id_2 = (select id from users WHERE username=$1)`
	rows, err := db.Query(query, username)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Printf("TEST %s:%T", username, username)
	defer rows.Close()

	for rows.Next() {
		var chat Chat
		if err := rows.Scan(&chat.ChatID, &chat.UserID1, &chat.UserID2, &chat.UserName1, &chat.UserName2); err != nil {
			log.Println(err)
			return nil, err
		}
		chats = append(chats, chat)
	}

	log.Println("чаты успешно получены:", chats)

	return chats, nil
}

func PostMessage(c echo.Context) error {
	db := db.Get()

	chatIDStr := c.Param("chat_id")
	userID := c.Param("user_id")
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id=$1", userID).Scan(&username)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("chat id: %s, user id: %s", chatIDStr, userID)
	if chatIDStr == "" || userID == "" {
		log.Println("не должно быть пустым")
		return c.String(http.StatusInternalServerError, "не должно быть пустым")
	}
	chatID, err := strconv.Atoi(chatIDStr) // Преобразование строки в int
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный формат chat_id")
	}
	log.Println("chat id(format):", chatID)

	content := c.FormValue("message")
	if content == "" {
		return c.String(http.StatusBadRequest, "Сообщение не может быть пустым")
	}

	// Создаем сообщение вручную
	msg := Message{
		ChatID:  chatID,
		UserID:  userID,
		Content: content,
	}

	if err := WriteMessageByID(msg); err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Ошибка при записи сообщения")
	}
	messanges, err := GetMessageByrID(chatID)
	if err != nil {
		log.Println(err)
		return err
	}
	return c.Render(http.StatusOK, "chatPage.html", messanges)
}
