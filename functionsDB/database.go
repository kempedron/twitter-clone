package functionsdb

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

var db *sql.DB
var err error

func InitDB() error {
	db, err = sql.Open("postgres", "user=postgres password=322 dbname=twitter sslmode=disable")
	if err != nil {
		log.Println(err)
		return err
	}
	return db.Ping()
}

type User struct {
	Id       int
	Username string
	Password string
}

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
	if c.Request().Method == http.MethodPost {
		inputUsername := c.FormValue("username")
		inputPassword := c.FormValue("password")

		if strings.TrimSpace(inputPassword) == "" || strings.TrimSpace(inputUsername) == "" {
			return c.String(http.StatusBadRequest, "поля не должны быть пустыми")
		}

		var storedPassword string
		err := db.QueryRow("SELECT user_password FROM users WHERE username = $1", inputUsername).Scan(&storedPassword)
		if err != nil {
			log.Println("Ошибка при получении пароля:", err)
			return c.String(http.StatusUnauthorized, "неверное имя пользователя")
		}

		if storedPassword != inputPassword {
			log.Println("Неверный пароль для пользователя:", inputUsername)
			return c.String(http.StatusUnauthorized, "неверный пароль")
		}

		// Устанавливаем куку
		cookie := new(http.Cookie)
		cookie.Name = "username"
		cookie.Value = inputUsername
		cookie.Expires = time.Now().Add(24 * time.Hour)
		cookie.Path = "/"
		c.SetCookie(cookie)

		// Работа с сессией
		session, _ := store.Get(c.Request(), "session")
		session.Values["username"] = inputUsername
		log.Println("Текущая сессия:", session.Values)

		session.Save(c.Request(), c.Response())

		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/home-page/%s", inputUsername))
	}

	return c.String(http.StatusMethodNotAllowed, "неверный метод запроса")
}

func SeeTweets(c echo.Context) error {
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
		var post Post
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

func RegisterNewUser(c echo.Context) error {
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

	return c.Redirect(http.StatusSeeOther, "/home-page")
}

func SearchUsers(c echo.Context) error {
	username := c.FormValue("username")
	log.Println("Получен юзернейм:", username)

	rows, err := db.Query("SELECT username FROM users WHERE username=$1", username)
	if err != nil {
		log.Printf("Ошибка при выполнении запроса: %v", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
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
	return c.Render(http.StatusOK, "ListUsersPage.html", users)
}

func Follow(c echo.Context) error {
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
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Пользователь не найден")
		}
		log.Println(err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
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
