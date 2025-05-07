package functionsdb

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"
	db "twitter/DataBase"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

type User struct {
	Id       int
	Username string
	Password string
}

var Store = sessions.NewCookieStore([]byte("secret-key"))

type Post struct {
	PostId       int
	AuthID       int
	Title        string
	Content      string
	PublicTime   string
	AuthUsername string
}

var store = sessions.NewCookieStore([]byte("secret-key"))

func LogIn(c echo.Context) error {
	db := db.Get()
	if c.Request().Method == http.MethodPost {
		inputUsername := c.FormValue("username")
		inputPassword := c.FormValue("password")

		if strings.TrimSpace(inputPassword) == "" || strings.TrimSpace(inputUsername) == "" {
			return c.String(http.StatusBadRequest, "Поля не должны быть пустыми")
		}

		var storedPassword string
		err := db.QueryRow("SELECT user_password FROM users WHERE username = $1", inputUsername).Scan(&storedPassword)
		if err != nil {
			log.Println("Ошибка при получении пароля:", err)
			return c.String(http.StatusUnauthorized, "Неверное имя пользователя")
		}

		if storedPassword != inputPassword {
			log.Println("Неверный пароль для пользователя:", inputUsername)
			return c.String(http.StatusUnauthorized, "Неверный пароль")
		}

		// Устанавливаем куку
		cookie := new(http.Cookie)
		cookie.Name = "username"
		cookie.Value = inputUsername
		cookie.Expires = time.Now().Add(24 * time.Hour)
		cookie.Path = "/"
		c.SetCookie(cookie)

		// Работа с сессией
		session, err := store.Get(c.Request(), "session")
		if err != nil {
			log.Printf("Ошибка при получении сессии: %v", err)
			return c.String(http.StatusInternalServerError, "Ошибка сервера")
		}

		session.Values["username"] = inputUsername
		log.Println("Текущая сессия перед сохранением:", session.Values)

		if err := session.Save(c.Request(), c.Response()); err != nil {
			log.Printf("Ошибка при сохранении сессии: %v", err)
			return c.String(http.StatusInternalServerError, "Ошибка сервера")
		}

		return c.Redirect(http.StatusSeeOther, "/home-page")
	}

	return c.String(http.StatusMethodNotAllowed, "Неверный метод запроса")
}

func CheckAuthorization(c echo.Context) error {
	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Printf("Ошибка при получении сессии: %v", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		log.Println("Пользователь не авторизован")
		return c.String(http.StatusUnauthorized, "необходимо войти")
	}

	log.Println("Пользователь авторизован:", username)
	return c.String(http.StatusOK, "Добро пожаловать, "+username)
}

func SeeTweets(c echo.Context) error {
	db := db.Get()

	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}

	// Извлекаем username из сессии
	username, ok := session.Values["username"].(string)
	log.Printf("Содержимое сессии: %v\n", session.Values)

	if !ok || username == "" {
		log.Println("Пользователь не аутентифицирован")
		return c.String(http.StatusUnauthorized, "необходимо войти")
	}

	rows, err := db.Query("SELECT post_id, user_id,post_title,post_content,TO_CHAR(created_at, 'DD Mon YYYY, HH24:MI') AS created_at FROM posts WHERE user_id = (SELECT id FROM users WHERE username = $1)", username)
	if err != nil {
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}

	defer rows.Close()
	var posts []Post

	for rows.Next() {
		if err := rows.Scan(&post.PostId, &post.AuthID, &post.Title, &post.Content, &post.PublicTime); err != nil {
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}
		posts = append(posts, post)

		err := rows.Err()
		if err != nil {
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}

	}
	return c.Render(http.StatusOK, "index.html", posts)

}

var post Post

func RegisterNewUser(c echo.Context) error {
	db := db.Get()

	newUserName := c.FormValue("newUserName")
	newUserPassword := c.FormValue("newUserPassword")
	result, err := db.Exec("INSERT INTO users(username, user_password) VALUES($1, $2) ON CONFLICT (username) DO NOTHING", newUserName, newUserPassword)
	if err != nil {
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	rowsAffect, err := result.RowsAffected()
	if err != nil {
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}

	if rowsAffect == 0 {
		return c.String(http.StatusConflict, "пользователь с таким именим уже зарегистрирован")
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

func SearchUsers(c echo.Context) error {
	db := db.Get()

	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Printf("Ошибка при получении сессии: %v", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	youUsername, ok := session.Values["username"].(string)
	if !ok || youUsername == "" {
		log.Println("Пользователь не авторизован")
		return c.String(http.StatusUnauthorized, "необходимо войти")
	}

	username := c.FormValue("username")
	log.Println("Получен юзернейм:", username)
	var storedUsername string
	err = db.QueryRow("SELECT username FROM users WHERE username=$1", username).Scan(&storedUsername)
	if err != nil {
		log.Printf("Ошибка при выполнении запроса: %v", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	var youID, userID int
	if err := db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID); err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}

	if err := db.QueryRow("SELECT id FROM users WHERE username=$1", youUsername).Scan(&youID); err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}
	chatID, err := GetChatForButton(userID, youID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = db.QueryRow(`
                INSERT INTO chats(user_id_1, user_id_2) 
                VALUES($1, $2) 
                RETURNING chat_id`,
				youID, userID).Scan(&chatID)

			if err != nil {
				log.Println("Ошибка при создании чата:", err)
				return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера при создании чата")
			}
			log.Println("Создан новый чат с ID:", chatID)
		} else {
			log.Println("Ошибка при получении chatID:", err)
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}
	}

	response := map[string]interface{}{
		"chatID":   chatID,
		"username": storedUsername,
	}

	return c.Render(http.StatusOK, "ListUsersPage.html", response)
}

func Follow(c echo.Context) error {
	db := db.Get()

	username2 := c.FormValue("username2")
	usernameCookie, err := c.Cookie("username")
	if err != nil {
		return c.String(http.StatusUnauthorized, "необходимо войти в систему")
	}
	username := usernameCookie.Value

	var id_us2 int
	var id_us1 int
	log.Println("user1", usernameCookie, "user2", username2)

	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username2).Scan(&id_us2)
	if err == sql.ErrNoRows {
		return c.String(http.StatusNotFound, "Пользователь не найден")
	}

	if err != nil {
		log.Println(err)

		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}
	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&id_us1)

	if err != nil {
		log.Println("ошибка", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}

	_, err = db.Exec("INSERT INTO followers(first_user_id,second_user_id) VALUES($1, $2) ON CONFLICT (first_user_id, second_user_id) DO NOTHING", id_us1, id_us2)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}

	return c.Redirect(http.StatusSeeOther, "/view-subscrives")
}

func ViewAllSubscribe(c echo.Context) error {
	db := db.Get()

	usernameCookie, err := c.Cookie("username")
	username := usernameCookie.Value
	if err != nil {
		return c.String(http.StatusUnauthorized, "необходимо войти в систему")
	}
	var user_id int
	_ = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&user_id)

	var idForName int
	err = db.QueryRow("SELECT second_user_id FROM followers WHERE first_user_id=$1", user_id).Scan(&idForName)
	if err != nil {
		log.Printf("Ошибка при выполнении запроса: %v", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	rows, err := db.Query("SELECT username FROM users WHERE id=$1", idForName)
	if err != nil {
		log.Println(err)
	}

	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		log.Println(user)
		if err := rows.Scan(&user.Username); err != nil {
			log.Printf("Ошибка при сканировании результата: %v", err)
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}
		users = append(users, user)
		err := rows.Err()
		if err != nil {
			return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
		}
	}

	if len(users) == 0 {
		return c.String(http.StatusNotFound, "Пользователь не найден")
	}
	log.Println(len(users))
	return c.Render(http.StatusOK, "viewSubscribes.html", users)

}

func CreateNewPost(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		log.Println("неверный метод")
		return c.String(http.StatusMethodNotAllowed, "неверный метод запроса")
	}
	db := db.Get()
	postTitle := c.FormValue("postTitle")
	postContent := c.FormValue("postContent")

	session, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сессии")
	}
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		return c.Redirect(http.StatusFound, "/login")
	}
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID)
	if err != nil {
		log.Println("error(db.go):", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка севера")
	}
	query := `INSERT INTO posts(user_id,post_title,post_content) VALUES($1,$2,$3)`
	_, err = db.Exec(query, userID, postTitle, postContent)
	if err != nil {
		log.Println("error in INSERT query:", err)
	}
	return c.Redirect(http.StatusSeeOther, "/home-page")
}

func CreateNewPostPage(c echo.Context) error {
	return c.File("templates/AddTweetForUser.html")
}

func GetChatForButton(userID1, userID2 int) (int, error) {
	db := db.Get()
	query := `SELECT chat_id FROM chats
	WHERE (user_id_1=$1 AND user_id_2=$2) OR (user_id_1=$2 AND user_id_2=$1)`
	var chatID int
	err := db.QueryRow(query, userID1, userID2).Scan(&chatID)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return chatID, nil
}
