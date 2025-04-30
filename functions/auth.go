package functions

import (
	"log"
	"net/http"
	functionsChat "twitter/chat"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

var store = sessions.NewCookieStore([]byte("secret-key"))

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := store.Get(c.Request(), "session")
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
	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка сессии")
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		log.Println("Пользователь не авторизован")
		return echo.NewHTTPError(http.StatusUnauthorized, "Пользователь не аутентифицирован")
	}

	chats, err := functionsChat.GetChatsByUsername(username)
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
