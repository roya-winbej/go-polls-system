package db

import (
	"log"

	"gopkg.in/mgo.v2"
)

var DB *mgo.Session

func Dialdb() error {
	var err error
	log.Println("Dialing mongodb: localhost")
	DB, err = mgo.Dial("localhost")
	return err
}

func Closedb() {
	DB.Close()
	log.Println("Database connection closed")
}
