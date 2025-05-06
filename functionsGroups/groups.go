package functionsGroups

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	db "twitter/DataBase"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var Store = sessions.NewCookieStore([]byte("secret-key"))

type Group struct {
	ID               int
	OwnerID          int
	GroupName        string
	GroupDescription string
}

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
	var groupID int
	err = db.QueryRow(`
    INSERT INTO groups(owner_id, group_name, group_description) 
    VALUES($1, $2, $3)
    RETURNING id`, userID, groupName, groupDescription).Scan(&groupID)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // Код ошибки unique_violation в PostgreSQL
				return c.String(http.StatusConflict, "группа с таким именем уже существует")
			}
		}

		log.Println("Ошибка при создании группы:", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}

	if groupID == 0 {
		return c.String(http.StatusInternalServerError, "не удалось создать группу")
	}

	_, err = db.Exec("INSERT INTO infoforgroups VALUES($1,$2)", groupID, userID)
	if err != nil {
		log.Println("error in line 65:", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}

	return c.Redirect(http.StatusSeeOther, "/home-page")

}

func ViewGroupsForUser(c echo.Context) error {
	db := db.Get()

	// Получаем сессию
	session, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сессии")
	}

	// Проверяем авторизацию
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		log.Println("Пользователь не авторизован")
		return c.Redirect(http.StatusFound, "/login")
	}

	// Получаем ID пользователя
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("тут пусто")
			return c.String(http.StatusNotFound, "Пользователь не найден")
		}
		log.Println("Ошибка при получении ID пользователя:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	// Получаем группы пользователя
	query := "SELECT id, owner_id, group_name, group_description FROM groups WHERE id=$1"
	rows, err := db.Query(query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("тут пусто")
			// Нет групп - возвращаем пустой список
			return c.Render(http.StatusOK, "ViewAllGroups.html", []Group{})
		}
		log.Println("Ошибка при запросе групп:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}
	defer rows.Close()

	// Собираем группы
	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.OwnerID, &group.GroupName, &group.GroupDescription); err != nil {
			log.Println("Ошибка при сканировании группы:", err)
			continue
		}
		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		log.Println("Ошибка после обработки групп:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	log.Printf("Найдено %d групп для пользователя %s", len(groups), username)
	return c.Render(http.StatusOK, "ViewGroupForUser.html", groups)
}

type ForGroupList struct {
	ID               int
	OwnerID          int
	GroupName        string
	GroupDescription string
	MemberCount      int
}

type Post struct {
	PostContent string
	PostTitle   string
	CreatedAt   string
	PostID      int
	GroupID     int
}

type ForGroupPage struct {
	ID               int
	OwnerID          int
	GroupName        string
	GroupDescription string
	MemberCount      int
	IsAdmin          bool
	Posts            []Post
}

func GetAllGroups(c echo.Context) error {
	db := db.Get()
	query := `select 
	g.id, g.owner_id,g.group_name,g.group_description, COUNT(i.user_id) as members_count
	FROM groups g
	LEFT JOIN infoforgroups i on g.id=i.group_id
	GROUP BY g.id, g.owner_id, g.group_name, g.group_description
	ORDER BY members_count ASC`
	rows, err := db.Query(query)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusOK, "ViewAllGroups.html", ForGroupList{})
		}
		log.Println("err for get groups in line 158:", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}
	var listGroups []ForGroupList
	defer rows.Close()
	for rows.Next() {
		var listGroup ForGroupList
		err := rows.Scan(&listGroup.ID, &listGroup.OwnerID, &listGroup.GroupName, &listGroup.GroupDescription, &listGroup.MemberCount)

		if err != nil {
			log.Println("err for scannig rows in line 166:", err)
			return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		log.Println(listGroup.MemberCount)

		listGroups = append(listGroups, listGroup)
	}
	return c.Render(http.StatusOK, "ViewAllGroups.html", listGroups)
}

func SubscribeOnGroup(c echo.Context) error {
	db := db.Get()
	groupID := c.Param("group-id")
	session, _ := Store.Get(c.Request(), "session")
	username, ok := session.Values["username"].(string)
	if !ok {
		return c.String(http.StatusUnauthorized, "Требуется авторизация")
	}
	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID); err != nil {
		log.Println("error in line 187:", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}

	_, err := db.Exec("INSERT INTO infoforgroups VALUES($1,$2)", groupID, userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // Код ошибки unique_violation в PostgreSQL
				return c.String(http.StatusConflict, "пользователь уже состоит в этой группе")
			}
		}
		log.Println("Ошибка при добавлении в группу:", err)
		return c.String(http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
	return c.String(http.StatusOK, "вы успешно подписались")
}

func FuncForViewGroup(c echo.Context) error {
	db := db.Get()
	groupID := c.Param("group-id")

	// Получение сессии и проверка авторизации
	session, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сессии")
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Получение ID пользователя
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Пользователь не найден")
		}
		log.Println("Ошибка при получении ID пользователя:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	// Проверка, является ли пользователь администратором
	var isAdmin bool
	err = db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM groups_admin 
            WHERE group_id = $1 AND admin_id = $2
        )`, groupID, userID).Scan(&isAdmin)
	if err != nil {
		log.Println("Ошибка проверки администратора:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	// Получение информации о группе
	var groupInfo ForGroupPage
	err = db.QueryRow(`
        SELECT 
            g.id, 
            g.owner_id,
            g.group_name,
            g.group_description,
            COUNT(DISTINCT i.user_id) as members_count
        FROM 
            groups g
        LEFT JOIN 
            infoforgroups i ON g.id = i.group_id
        WHERE 
            g.id = $1
        GROUP BY 
            g.id, g.owner_id, g.group_name, g.group_description`, groupID).Scan(
		&groupInfo.ID,
		&groupInfo.OwnerID,
		&groupInfo.GroupName,
		&groupInfo.GroupDescription,
		&groupInfo.MemberCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Группа не найдена")
		}
		log.Println("Ошибка получения информации о группе:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}
	groupInfo.IsAdmin = isAdmin

	// Получение постов группы
	rows, err := db.Query(`
        SELECT 
            post_id, 
            post_title, 
            post_content, 
            created_at
        FROM 
            group_post
        WHERE 
            group_id = $1
        ORDER BY 
            created_at DESC`, groupID)
	if err != nil {
		log.Println("Ошибка получения постов:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}
	defer rows.Close()

	// Сбор постов
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.PostID, &post.PostTitle, &post.PostContent, &post.CreatedAt); err != nil {
			log.Println("Ошибка сканирования поста:", err)
			continue
		}
		post.GroupID = groupInfo.ID
		groupInfo.Posts = append(groupInfo.Posts, post)
	}

	if err = rows.Err(); err != nil {
		log.Println("Ошибка после обработки постов:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сервера")
	}

	err = c.Render(http.StatusOK, "PageForGroup.html", groupInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func AddPostInGroup(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.String(http.StatusMethodNotAllowed, "метод не разрешен")
	}
	db := db.Get()


	// Логируем заголовки запроса
	log.Println("Заголовки запроса:", c.Request().Header)

	// Логируем тело запроса
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Println("Ошибка чтения тела запроса:", err)
	} else {
		log.Println("Тело запроса:", string(body))
		// Восстанавливаем тело для дальнейшего чтения
		c.Request().Body = io.NopCloser(bytes.NewReader(body))
	}

	// Получаем значения из формы
	postTitle := c.FormValue("postTitle")
	postContent := c.FormValue("postContent")
	
	groupID := c.Param("group-id")

	log.Printf("Полученные данные: title='%s', content='%s', groupID='%s'",
		postTitle, postContent, groupID)

	session, err := Store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Ошибка при получении сессии:", err)
		return c.String(http.StatusInternalServerError, "Ошибка сессии")
	}
	log.Println("title:", postTitle, "content:", postContent)

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		return c.Redirect(http.StatusFound, "/login")
	}
	var userID int
	var isAdmin bool
	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID)
	if err != nil {
		log.Println("error in line 350(get id for user):", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM groups_admin WHERE admin_id=$1)", userID).Scan(&isAdmin)
	if err != nil {
		log.Println("err in line 363(check-admin query):", err)
		return c.String(http.StatusInternalServerError, "ошибка на стороне сервера")
	}
	if !isAdmin {
		return c.String(http.StatusInternalServerError, "извините,похоже у вас недостаточно прав для добавления постов")
	}
	_, err = db.Exec(`INSERT INTO 
	group_post(post_title, post_content, group_id, creator_id)
	VALUES($1,$2,$3,$4)`, postTitle, postContent, groupID, userID)
	if err != nil {
		log.Println("err in line 373(add post):", err)
		return c.String(http.StatusInternalServerError, "внутрення ошибка на стороне сервера")
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/view-group/%s", groupID))
}

func AddPostInGroupPage(c echo.Context) error {

	groupID := c.Param("group-id")
	log.Println(groupID)

	data := map[string]interface{}{
		"groupID": groupID,
	}
	err := c.Render(http.StatusOK, "AddPostPage.html", data)
	if err != nil {
		log.Println("rendering error in line 382:", err)
		return err
	}
	return nil
}
