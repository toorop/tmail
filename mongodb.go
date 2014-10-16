package main

import (
	"gopkg.in/mgo.v2"
)

// getMgoSession returns a valid mgo session
func getMgoSession() (*mgo.Session, error) {
	var err error
	if mgoSession == nil || mgoSession.Ping() == nil {
		mgoSession, err = mgo.Dial(Config.StringDefault("mongo.url", "localhost"))
		if err != nil {
			return nil, err
		}
		// Optional. Switch the session to a monotonic behavior.
		mgoSession.SetMode(mgo.Monotonic, true)
	}
	return mgoSession.Clone(), nil
}
