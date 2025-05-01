package functionsGroups

import (
	"log"
	"net/http"
	db "twitter/DataBase"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

var Store = sessions.NewCookieStore([]byte("secret-key"))

func CreateNewGroup(c echo.Context) error {
	db := db.Get()

	groupName := c.FormValue("group-name")
	groupDescription := c.FormValue("group-description")
	log.Println(groupName, groupDescription)

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
	var userID int
	err = db.QueryRow("SELECT id from users WHERE username=$1", username).Scan(&userID)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("Запрос к базе данных для получения userID для пользователя:", username, userID)

	log.Println("Создание группы с owner_id:", userID, "group_name:", groupName, "group_description:", groupDescription)

	query := `INSERT INTO groups(owner_id,group_name,group_description) VALUES($1, $2, $3)`
	result, err := db.Exec(query, userID, groupName, groupDescription)
	if err != nil {
		log.Println(1, err)
		return err
	}
	log.Println(4)
	rowsAffect, err := result.RowsAffected()
	if err != nil {
		log.Println(2, err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	log.Println(5)

	if rowsAffect == 0 {
		return c.String(http.StatusConflict, "пользователь с таким именим уже зарегистрирован")
	}
	log.Println(6)

	return c.Redirect(http.StatusSeeOther, "/home-page")

}

func ViewAllGroup(c echo.Context) error {
	db := db.Get()

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
	var userID int

	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID)
	if err != nil {
		log.Println(err)
		return err
	}
	var groupID int
	err = db.QueryRow("select group_id from infoForGroups WHERE user_id=$1").Scan(&groupID)
	if err != nil {
		log.Println(err)
		return err
	}
	query := `select id,owner_id,group_name,group_description from groups where id=$1`
	rows, err := db.Query(query, groupID)
	if err != nil {
		log.Println(err)
		return err
	}

}
