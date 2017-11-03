package database

import (
	"gopkg.in/mgo.v2"
)

func Persist(car *Car) error {
	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("28car").C("cars")
	err = c.Insert(car)
	return err
}
