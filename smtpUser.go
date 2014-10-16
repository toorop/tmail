package main

import (
	//"gopkg.in/mgo.v2"
	"code.google.com/p/go.crypto/bcrypt"
	"errors"
	"gopkg.in/mgo.v2/bson"
)

type smtpUser struct {
	Login     string
	Passwd    string
	AuthRelay bool
}

// NewSmtpUser return a new authentificated smtp user
func NewSmtpUser(login, passwd string) (user *smtpUser, err error) {
	// verification des entres
	if len(login) == 0 || len(passwd) == 0 {
		err := errors.New("login or passwd is empty")
		return nil, err
	}

	// On recupere un session mgo
	s, err := getMgoSession()
	if err != nil {
		return
	}
	defer s.Close()
	c := s.DB(Config.StringDefault("mongo.db", "tmail")).C("smtpusers")
	err = c.Find(bson.M{"login": login}).One(&user)
	if err != nil {
		return
	}
	// Encoding passwd
	/*hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	TRACE.Println(string(hashed), err)*/

	// Check passwd
	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(passwd))
	return
}

// check if user can relay throught this server
// TODO je pense qy'il faudrait mettre le destinataires pour les limitation par destinataion
func (s *smtpUser) canUseSmtp() (bool, error) {
	return true, nil
}
