package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	functionsChat "twitter/chat"
	"twitter/functions"
	functionsdb "twitter/functionsDB"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

type TemplateRenderer struct {
	templates *template.Template
}

// Render метод для рендеринга шаблонов
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	if err := functionsdb.InitDB(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	if err := functionsChat.InitDB(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных(chat): %v", err)
	}
	fmt.Println("start twitter")
	e := echo.New()
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Use(AuthMiddleware)

	e.GET("/", functions.StartPage)
	e.GET("/home-page", functionsdb.SeeTweets, AuthMiddleware)
	e.GET("/login-page", functions.LoginPage)
	e.POST("/login-method", functionsdb.LogIn)
	e.GET("/register-page", functions.RegisterPage)
	e.POST("/register-new-user", functionsdb.RegisterNewUser)
	e.GET("/search-users", functions.PageForSearch) // Страница поиска пользователей
	e.POST("/search-method", functionsdb.SearchUsers)
	e.POST("/follow-method", functionsdb.Follow)
	e.GET("/follow-page", functions.FollowPage)
	e.GET("/view-subscrives", functionsdb.ViewAllSubscribe)
	e.GET("/api/messages/:chat_id", functionsChat.GetMessages) // Получение сообщений по chat_id
	e.POST("/api/messages", functionsChat.PostMessage)

	if err := e.Start("127.0.0.1:8080"); err != nil {
		log.Println(err)
		log.Fatal(err)

	}
}

var store = sessions.NewCookieStore([]byte("secret-key"))

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := store.Get(c.Request(), "session")
		if err != nil {
			log.Println("Ошибка при получении сессии:", err)
			return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
		}

		username, ok := session.Values["username"].(string)

		session.Values["username"] = username
		if err := session.Save(c.Request(), c.Response()); err != nil {
		}

		if !ok || username == "" {
			return c.String(http.StatusUnauthorized, "необходимо войти")
		}

		return next(c)
	}
}

func WebSocketHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}

		// Преобразование []byte в string
		messageString := string(msg)

		// Широковещательная рассылка сообщения всем подключенным клиентам
		mu.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(messageString)); err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
	return nil
}
