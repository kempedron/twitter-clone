package functions

import "github.com/labstack/echo/v4"

func MainPage(c echo.Context) error {
	return c.File("templates/index.html")
}

func LoginPage(c echo.Context) error {
	return c.File("templates/LoginPage.html")
}

func StartPage(c echo.Context) error {
	return c.File("templates/StartPage.html")
}

func RegisterPage(c echo.Context) error {
	return c.File("templates/RegisterPage.html")
}

func SearchUsersPage(c echo.Context) error {
	return c.File("templates/ListUsersPage.html")
}

func PageForSearch(c echo.Context) error {
	return c.File("templates/PageForSearch.html")
}
func FollowPage(c echo.Context) error {
	return c.File("templates/PageForFollow.html")
}

func ViewSubscribes(c echo.Context) error {
	return c.File("templates/viewSubscribes.html")
}

func CreateGroupPage(c echo.Context) error {
	return c.File("templates/NewGroupPage.html")
}

