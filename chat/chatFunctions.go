package functionsChat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	db "twitter/DataBase"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

var clients = make(map[string]map[*websocket.Conn]bool) // map[chatID]map[conn]bool
var mu sync.Mutex
var Store = sessions.NewCookieStore([]byte("secret-key"))

type Message struct {
	ID        int    `json:"ID"`
	ChatID    int    `json:"chat_id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	CreatedAt string `json:"send_time"`
}

// структура для чата
type Chat struct {
	ChatID    int    `json:"chat_id"`
	UserID1   int    `json:"user_id1"`
	UserID2   int    `json:"user_id2"`
	UserName1 string `json:"username1"`
	UserName2 string `json:"username2"`
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

	data := ChatPageData{
		ChatID:   chatID,
		Messages: []Message{},
	}

	// Получаем user_id из первого сообщения в чате (если есть)
	var userID int
	err := db.QueryRow(`
        SELECT user_id FROM messages 
        WHERE chat_id = $1 
        LIMIT 1`, chatID).Scan(&userID)

	// Обрабатываем случай, когда чат существует, но без сообщений
	if err == sql.ErrNoRows {
		return data, nil
	}
	if err != nil {
		log.Printf("Error getting user_id: %v", err)
		return ChatPageData{}, fmt.Errorf("error getting chat info: %v", err)
	}

	// Получаем username (если user_id найден)
	var username string
	err = db.QueryRow(`
        SELECT username FROM users 
        WHERE id = $1`, userID).Scan(&username)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting username: %v", err)
		return ChatPageData{}, fmt.Errorf("error getting user info: %v", err)
	}

	// Получаем все сообщения чата
	rows, err := db.Query(`
        SELECT 
            m.id, 
            m.chat_id, 
            m.user_id, 
            u.username, 
            m.content, 
            TO_CHAR(m.created_at, 'DD Mon YYYY HH24:MI:SS') AS formatted_time 
        FROM messages m
        JOIN users u ON m.user_id = u.id
        WHERE m.chat_id = $1 
        ORDER BY m.created_at`, chatID)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error querying messages: %v", err)
		return ChatPageData{}, fmt.Errorf("error getting messages: %v", err)
	}
	defer rows.Close()

	// Собираем сообщения
	for rows.Next() {
		var msg Message
		var userID int // Используем int для сканирования из БД

		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&userID,
			&msg.Username,
			&msg.Content,
			&msg.CreatedAt,
		)
		log.Println("id:", userID)

		if err != nil {
			log.Printf("Error scanning message: %v", err)
			continue // Пропускаем ошибочные записи
		}

		// Конвертируем userID в строку для структуры Message
		msg.UserID = strconv.Itoa(userID)
		data.Messages = append(data.Messages, msg)
	}

	// Проверяем ошибки итерации
	if err = rows.Err(); err != nil {
		log.Printf("Rows iteration error: %v", err)
		return ChatPageData{}, fmt.Errorf("error processing messages: %v", err)
	}

	// Устанавливаем UserID для ChatPageData (как строку)
	data.UserID = strconv.Itoa(userID)

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
	db := db.Get()

	// Получаем chat_id из URL
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный ID чата")
	}

	// Получаем user_id из сессии
	session, _ := Store.Get(c.Request(), "session")
	username, ok := session.Values["username"].(string)
	if !ok {
		return c.String(http.StatusUnauthorized, "Требуется авторизация")
	}

	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		return c.String(http.StatusNotFound, "Пользователь не найден")
	}

	// Получаем данные чата
	chatData, err := GetMessageByrID(chatID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Ошибка загрузки чата")
	}

	// Устанавливаем UserID для шаблона
	chatData.UserID = strconv.Itoa(userID)

	return c.Render(http.StatusOK, "chatPage.html", chatData)
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

// functionsChat/functionsChat.go

func PostMessage(c echo.Context) error {
	db := db.Get()
	if db == nil {
		return c.String(http.StatusInternalServerError, "Database connection error")
	}

	// Получаем параметры
	chatIDStr := c.Param("chat_id")
	userIDStr := c.Param("user_id")
	content := strings.TrimSpace(c.FormValue("message"))

	// Валидация и преобразование параметров
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil || chatID <= 0 {
		return c.String(http.StatusBadRequest, "Неверный ID чата")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		return c.String(http.StatusBadRequest, "Неверный ID пользователя")
	}

	// Получаем username
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)
	if err != nil {
		return c.String(http.StatusNotFound, "Пользователь не найден")
	}

	// Создаем сообщение
	msg := Message{
		ChatID:   chatID,
		UserID:   strconv.Itoa(userID),
		Username: username,
		Content:  content,
	}

	// Сохраняем в БД
	if err := WriteMessageByID(msg); err != nil {
		return c.String(http.StatusInternalServerError, "Ошибка сохранения сообщения")
	}

	// Формируем JSON для отправки через WebSocket
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Ошибка маршалинга сообщения: %v", err)
	} else {
		// Отправляем через WebSocket всем подписчикам этого чата
		mu.Lock()
		for conn := range clients[chatIDStr] {
			if err := conn.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
				conn.Close()
				delete(clients[chatIDStr], conn)
			}
		}
		mu.Unlock()
	}

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/api/messages/%d", chatID))
}

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := Store.Get(c.Request(), "session")
		if err != nil {
			log.Println("Ошибка при получении сессии:", err)
			return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
		}

		log.Println("Содержимое сессии:", session.Values)

		username, ok := session.Values["username"].(string)
		log.Println(username)
		if !ok {
			log.Println("Пользователь не авторизован")
			return c.String(http.StatusUnauthorized, "Пользователь не авторизован")
		}

		if err := session.Save(c.Request(), c.Response()); err != nil {
			log.Println("Ошибка при сохранении сессии:", err)
			return c.String(http.StatusInternalServerError, "Ошибка сервера")
		}

		return next(c)
	}
}

func RecoverMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Паника: %v", r)
				c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
			}
		}()
		return next(c)
	}
}

func GetChats(c echo.Context) error {
	session, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка сессии")
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		log.Println("Пользователь не авторизован")
		return echo.NewHTTPError(http.StatusUnauthorized, "Пользователь не аутентифицирован")
	}

	chats, err := GetChatsByUsername(username)
	if err != nil {
		log.Println("Ошибка при получении чатов:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка при получении чатов")
	}

	log.Println("Чаты успешно получены:", chats)

	// Проверка структуры данных перед рендерингом
	log.Println("Данные для рендеринга:", chats)

	err = c.Render(http.StatusOK, "allChats.html", chats)
	if err != nil {
		log.Println("Ошибка рендеринга:", err)
		log.Println(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка рендеринга шаблона")
	}

	return nil
}

