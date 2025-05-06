package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	db "twitter/DataBase"
	functionsChat "twitter/chat"
	"twitter/functions"
	functionsdb "twitter/functionsDB"
	"twitter/functionsGroups"

	"github.com/labstack/echo/v4"
)

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

	// Публичные маршруты (без middleware)
	public := e.Group("")
	public.GET("/", functions.StartPage)
	public.GET("/login", functions.LoginPage)
	public.POST("/login", functionsdb.LogIn)
	public.GET("/register", functions.RegisterPage)
	public.POST("/register", functionsdb.RegisterNewUser)

	// Приватные маршруты (с middleware)
	private := e.Group("")
	private.Use(functionsChat.AuthMiddleware)
	private.Use(functionsChat.RecoverMiddleware)

	private.GET("/home-page", functionsdb.SeeTweets)
	private.GET("/search-users", functions.PageForSearch) // Страница поиска пользователей
	private.POST("/search-method", functionsdb.SearchUsers)
	private.POST("/follow-method", functionsdb.Follow)
	private.GET("/follow-page", functions.FollowPage)
	private.GET("/view-subscrives", functionsdb.ViewAllSubscribe)
	private.GET("/api/messages/:chat_id", functionsChat.GetMessages)
	private.POST("/api/messages/:chat_id/user/:user_id", functionsChat.PostMessage)
	private.GET("/get-chats", functionsChat.GetChats)
	private.POST("/create-new-group", functionsGroups.CreateNewGroup)
	private.GET("/create-new-group", functions.CreateGroupPage)
	private.GET("/get-groups-for-user", functionsGroups.ViewGroupsForUser)
	private.GET("/get-all-groups", functionsGroups.GetAllGroups)
	private.GET("/view-group/:group-id", functionsGroups.FuncForViewGroup)
	private.POST("/subscribe/group/:group-id", functionsGroups.SubscribeOnGroup)
	private.POST("/add-post-in-group/:group-id", functionsGroups.AddPostInGroup)
	private.GET("/add-post-in-group/:group-id", functionsGroups.AddPostInGroupPage)
	private.GET("/create-new-post", functionsdb.CreateNewPostPage)
	private.POST("/create-new-post", functionsdb.CreateNewPost)

	if err := e.Start("127.0.0.1:8080"); err != nil {
		log.Println(err)
		log.Fatal(err)
	}
}

// func WebSocketHandler(c echo.Context) error {
// 	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
// 	if err != nil {
// 		log.Println("Ошибка при обновлении соединения:", err)
// 		return c.String(http.StatusInternalServerError, "Ошибка при подключении WebSocket")
// 	}
// 	defer conn.Close()

// 	// Получаем chat_id из query параметров
// 	chatID := c.QueryParam("chat_id")
// 	if chatID == "" {
// 		log.Println("chat_id is required")
// 		return c.String(http.StatusBadRequest, "chat_id is required")
// 	}

// 	// Регистрируем соединение
// 	mu.Lock()
// 	if clients[chatID] == nil {
// 		clients[chatID] = make(map[*websocket.Conn]bool)
// 	}
// 	clients[chatID][conn] = true
// 	mu.Unlock()

// 	// Обработка входящих сообщений
// 	for {
// 		_, msg, err := conn.ReadMessage()
// 		if err != nil {
// 			log.Println("Ошибка чтения сообщения:", err)
// 			mu.Lock()
// 			delete(clients[chatID], conn)
// 			mu.Unlock()
// 			break
// 		}

// 		// Обработка и сохранение сообщения
// 		handleIncomingMessage(chatID, msg, conn)
// 	}
// 	return nil
// }

// func handleIncomingMessage(chatID string, msg []byte, sender *websocket.Conn) {
// 	// Здесь можно добавить логику сохранения сообщения в БД
// 	// и рассылки его всем участникам чата

// 	mu.Lock()
// 	defer mu.Unlock()

// 	for conn := range clients[chatID] {
// 		if conn != sender { // Отправляем всем, кроме отправителя
// 			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
// 				conn.Close()
// 				delete(clients[chatID], conn)
// 			}
// 		}
// 	}
// }

// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

// var clients = make(map[string]map[*websocket.Conn]bool) // map[chatID]map[conn]bool
// var mu sync.Mutex
