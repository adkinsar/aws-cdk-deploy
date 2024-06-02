package app

import (
	"lambda-func/api"
	"lambda-func/database"
)

type App struct {
	ApiHandler api.ApiHandler
}

func NewApp() App {

	db := database.NewDynamoDBClient()
	return App{
		ApiHandler: api.NewApiHandler(db),
	}
}
