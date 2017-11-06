package database

import (
	"gopkg.in/mgo.v2"
)

type MyMongo struct {
	host string
}

func New(host string) *MyMongo {
	return &MyMongo{
		host: host,
	}
}

func (myMongo *MyMongo) Persist(car *Car) error {
	session, err := mgo.Dial(myMongo.host)
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
