package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	db "twitter/DataBase"
	functionsChat "twitter/chat"
	"twitter/functions"
	functionsdb "twitter/functionsDB"

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
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		return err // Возвращаем ошибку, если рендеринг не удался
	}
	return nil
}

func main() {
	if err := db.Init("user=postgres password=322 dbname=twitter sslmode=disable"); err != nil {
		log.Println("ошибка инициализации БД", err)
		log.Fatal(err)
	}
	fmt.Println("start twitter")
	e := echo.New()
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Use(functions.AuthMiddleware)
	e.Use(functions.RecoverMiddleware)

	e.GET("/", functions.StartPage)
	e.GET("/home-page", functionsdb.SeeTweets)
	e.GET("/login-page", functions.LoginPage)
	e.POST("/login-method", functionsdb.LogIn)
	e.GET("/register-page", functions.RegisterPage)
	e.POST("/register-new-user", functionsdb.RegisterNewUser)
	e.GET("/search-users", functions.PageForSearch) // Страница поиска пользователей
	e.POST("/search-method", functionsdb.SearchUsers)
	e.POST("/follow-method", functionsdb.Follow)
	e.GET("/follow-page", functions.FollowPage)
	e.GET("/view-subscrives", functionsdb.ViewAllSubscribe)
	e.GET("/api/messages/:chat_id", functionsChat.GetMessages)
	e.POST("/chat/:chat_id/user/:user_id/message", functionsChat.PostMessage)
	e.GET("/get-chats", functions.GetChats)

	if err := e.Start("127.0.0.1:8080"); err != nil {
		log.Println(err)
		log.Fatal(err)

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
