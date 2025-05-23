package functionsChat

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	db "twitter/DataBase"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

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

		err := rows.Scan(&msg.ID, &msg.ChatID, &userID, &msg.Username, &msg.Content, &msg.CreatedAt)

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

func WriteMessageByIDForGCH(msg Message) error {
	db := db.Get()

	if db == nil {
		return errors.New("database connection is not initialized")
	}

	query := `INSERT INTO messages_gch(chat_id, user_id, content) VALUES ($1, $2, $3) RETURNING id`
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
			return nil, err
		}
		if chat.UserName2 == username {
			saveName2 := chat.UserName2
			chat.UserName2 = chat.UserName1
			chat.UserName1 = saveName2

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

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/api/messages/%d", chatID))
}

func PostMessageForGCH(c echo.Context) error {
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
	log.Println(1)
	if err != nil || chatID <= 0 {
		log.Printf("error chatID for post GCH func.Err: %s", err)
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
	if err := WriteMessageByIDForGCH(msg); err != nil {
		return c.String(http.StatusInternalServerError, "Ошибка сохранения сообщения")
	}

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/view-chat-group/%d", chatID))
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

func CheckUserInChatMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Получаем chat_id из URL
		chatID := c.Param("chat_id")
		if chatID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Не указан ID чата",
			})
		}

		sess, err := Store.Get(c.Request(), "session")
		if err != nil {
			log.Println("Ошибка сессии:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Ошибка сервера",
			})
		}

		username, ok := sess.Values["username"].(string)
		if !ok || username == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Требуется авторизация",
			})
		}

		// Проверяем доступ к конкретному чату
		var exists bool
		query := `
            SELECT EXISTS(
                SELECT 1 FROM chats 
                WHERE chat_id = $1 AND 
                      (user_id_1 = (SELECT id FROM users WHERE username = $2) OR 
                       user_id_2 = (SELECT id FROM users WHERE username = $2))
            )`

		err = db.Get().QueryRow(query, chatID, username).Scan(&exists)
		if err != nil {
			log.Println("Ошибка БД:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Ошибка проверки доступа",
			})
		}

		if !exists {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "У вас нет доступа к этому чату",
			})
		}

		return next(c)
	}
}

type GroupChat struct {
	GroupChatID   int
	MemberID      int
	GroupChatName string
}

func ViewAllChatGroup(c echo.Context) error {
	db := db.Get()

	sess, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка сессии:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Ошибка сервера",
		})
	}

	username, ok := sess.Values["username"].(string)
	if !ok || username == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Требуется авторизация",
		})
	}
	query := `SELECT g.group_chat_id, g.member_id, info.chat_group_name
	FROM group_chat g
	INNER JOIN info_for_chatgroup info ON g.group_chat_id = info.chat_group_id
	WHERE g.member_id = (SELECT id FROM users WHERE username=$1)`
	rows, err := db.Query(query, username)
	if err != nil {
		log.Printf("ошибка при получении групповых чатов: %s", err)
		return c.String(http.StatusInternalServerError, "ошибка при получении групповых-чатов")
	}
	defer rows.Close()

	var groupChats []GroupChat
	for rows.Next() {
		var groupChat GroupChat
		err := rows.Scan(&groupChat.GroupChatID, &groupChat.MemberID, &groupChat.GroupChatName)
		if err != nil {
			log.Printf("ошибка при сканировании данных для групповых чатов: %s", err)
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}
		groupChats = append(groupChats, groupChat)
	}
	//return c.String(http.StatusOK,"322")
	return c.Render(http.StatusOK, "ViewAllChatGroups.html", groupChats)

}

type DataForGroupChats struct {
	ChatID   int
	UserID   string
	Messages []Message
	Members  []string
}

func GetMessageByrIDForGCH(chatID int) (DataForGroupChats, error) {
	db := db.Get()
	if db == nil {
		return DataForGroupChats{}, errors.New("database connection is not initialized")
	}

	data := DataForGroupChats{
		ChatID:   chatID,
		Messages: []Message{},
		Members:  []string{},
	}

	// Получаем user_id из первого сообщения в чате (если есть)
	var userID int
	err := db.QueryRow(`
        SELECT user_id FROM messages_gch
        WHERE chat_id = $1 
        LIMIT 1`, chatID).Scan(&userID)

	// Обрабатываем случай, когда чат существует, но без сообщений
	if err == sql.ErrNoRows {
		return data, nil
	}
	if err != nil {
		log.Printf("Error getting user_id: %v", err)
		return DataForGroupChats{}, fmt.Errorf("error getting chat info: %v", err)
	}

	// Получаем username (если user_id найден)
	var username string
	err = db.QueryRow(`
        SELECT username FROM users 
        WHERE id = $1`, userID).Scan(&username)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting username: %v", err)
		return DataForGroupChats{}, fmt.Errorf("error getting user info: %v", err)
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
        FROM messages_gch m
        JOIN users u ON m.user_id = u.id
        WHERE m.chat_id = $1 
        ORDER BY m.created_at`, chatID)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error querying messages: %v", err)
		return DataForGroupChats{}, fmt.Errorf("error getting messages: %v", err)
	}
	defer rows.Close()

	// Собираем сообщения
	for rows.Next() {
		var msg Message
		var userID int // Используем int для сканирования из БД

		err := rows.Scan(&msg.ID, &msg.ChatID, &userID, &msg.Username, &msg.Content, &msg.CreatedAt)

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
		return DataForGroupChats{}, fmt.Errorf("error processing messages: %v", err)
	}

	// Устанавливаем UserID для ChatPageData (как строку)
	data.UserID = strconv.Itoa(userID)

	query := `
		SELECT 
			u.username
		FROM users u
		JOIN group_chat gch ON u.id = gch.member_id
		WHERE gch.group_chat_id=$1`

	rows, err = db.Query(query, chatID)
	if err != nil {
		log.Printf("error in sql-qury for get []users: %s", err)
		return DataForGroupChats{}, fmt.Errorf("error getting usernames: %s", err)
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var user string
		err := rows.Scan(&user)
		if err != nil {
			log.Printf("error in scan sql row for []users: %s", err)
			continue
		}
		users = append(users, user)
	}
	data.Members = users
	return data, nil
}

func GetMessagesForGCH(c echo.Context) error {
	db := db.Get()

	// Получаем chat_id из URL
	chatIDStr := c.Param("chat-group-id")
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Неверный ID чата")
	}

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
	chatData, err := GetMessageByrIDForGCH(chatID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "ошибка при получении сообщений")
	}

	// Устанавливаем UserID для шаблона
	chatData.UserID = strconv.Itoa(userID)
	if err := c.Render(http.StatusOK, "chatPageForGroupChat.html", chatData); err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Ошибка рендеринга страницы")
	}
	return nil
}

func AddToChatGroup(c echo.Context) error {
	if c.Request().Method == http.MethodPost {
		db := db.Get()
		groupChatID := c.Param("chat-group-id")
		log.Println(groupChatID)
		GChatID, err := strconv.Atoi(groupChatID)
		if err != nil {
			log.Printf("error in AddToChatGroup(convertion groupChatID): %s", err)
			return c.String(http.StatusInternalServerError, "ошибка на строне сервера")
		}
		usernameForAdd := c.FormValue("usernameForAdd")
		log.Println(groupChatID, usernameForAdd)

		if groupChatID == "" || usernameForAdd == "" {
			log.Printf("error(no get valeus): %s", err)
			return c.String(http.StatusInternalServerError, "поля не должны быть пустыми")
		}
		query := `INSERT INTO group_chat(group_chat_id, member_id) VALUES($1,(select id from users WHERE username=$2))`
		_, err = db.Exec(query, GChatID, usernameForAdd)
		if err != nil {
			if pgErr, ok := err.(*pq.Error); ok {
				// Конкретная проверка на нарушение уникальности первичного ключа
				if pgErr.Code == "23505" && pgErr.Constraint == "group_chat_pkey" {
					return c.JSON(http.StatusInternalServerError, "пользователь уже состоит в чате")
				}
			}
			log.Printf("error in AddToChatGroup: %s", err)
			return c.String(http.StatusInternalServerError, "ошибка сервера")
		}

		return c.String(http.StatusOK, fmt.Sprintf("вы успешно пригласили пользователя %s", usernameForAdd))
	}
	return c.JSON(http.StatusMethodNotAllowed, map[string]string{
		"error": "NotAllowedRequesMethod",
	})
}

func AddToChatGroupPage(c echo.Context) error {
	ID := c.Param("chat-group-id")
	data := map[string]string{
		"GroupChatID": ID,
	}
	return c.Render(http.StatusOK, "AddUserToChat.html", data)
}
