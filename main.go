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
	chat := e.Group("")
	chat.Use(functionsChat.CheckUserInChatMiddleware)
	chat.GET("/api/messages/:chat_id", functionsChat.GetMessages)
	chat.POST("/api/messages/:chat_id/user/:user_id", functionsChat.PostMessage)

	private := e.Group("")
	private.Use(functionsChat.AuthMiddleware)
	private.Use(functionsChat.RecoverMiddleware)

	private.POST("/delete-tweet/:tweet-id", functionsdb.DeleteTweet)
	private.POST("/delete-group-post/:group-id/:post-id", functionsGroups.DeletePost)
	private.GET("/home-page", functionsdb.SeeTweets)
	private.GET("/search-users", functions.PageForSearch) // Страница поиска пользователей
	private.POST("/search-method", functionsdb.SearchUsers)
	private.GET("/view-subscrives", functionsdb.ViewAllSubscribe)
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
	private.GET("/chat-groups", functionsChat.ViewAllChatGroup)
	private.GET("/view-chat-group/:chat-group-id", functionsChat.GetMessagesForCHatGroup)

	if err := e.Start("127.0.0.1:8080"); err != nil {
		log.Println(err)
		log.Fatal(err)
	}
}

// CheckUserInChatMiddleware проверяет, есть ли пользователь в чате
