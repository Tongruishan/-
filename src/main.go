package main

import (
	_ "pygHouse/src/routers"
	"github.com/astaxie/beego"
	_"pygHouse/src/models"
	_"github.com/go-sql-driver/mysql"
)

func main() {
	beego.Run()
}

