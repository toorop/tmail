package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"github.com/toorop/tmail/api"
)

// usersGetAll return all users
func queueGetMessages(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	messages, err := api.QueueGetMessages()
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message in queue", err.Error())
		return
	}
	js, err := json.Marshal(messages)
	if err != nil {
		httpWriteErrorJson(w, 500, "JSON encondig failed", err.Error())
		return
	}
	httpWriteJson(w, js)
}

// queueGetMessage get a message by ID
func queueGetMessage(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	msgIdStr := httpcontext.Get(r, "params").(httprouter.Params).ByName("id")
	msgIdInt, err := strconv.ParseInt(msgIdStr, 10, 64)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message id", err.Error())
		return
	}

	m, err := api.QueueGetMessage(msgIdInt)
	if err == gorm.ErrRecordNotFound {
		httpWriteErrorJson(w, 404, "no such message "+msgIdStr, "")
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message "+msgIdStr, err.Error())
		return
	}
	js, err := json.Marshal(m)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message "+msgIdStr, err.Error())
		return
	}
	httpWriteJson(w, js)
}

// queueDiscardMessage  discard a message (delete without bouncing)
func queueDiscardMessage(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	msgIdStr := httpcontext.Get(r, "params").(httprouter.Params).ByName("id")
	msgIdInt, err := strconv.ParseInt(msgIdStr, 10, 64)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message id", err.Error())
		return
	}
	err = api.QueueDiscardMsg(msgIdInt)
	if err == gorm.ErrRecordNotFound {
		httpWriteErrorJson(w, 404, "no such message "+msgIdStr, "")
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to discard message "+msgIdStr, err.Error())
		return
	}
}

// queueBounceMessage  bounce a message
func queueBounceMessage(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	msgIdStr := httpcontext.Get(r, "params").(httprouter.Params).ByName("id")
	msgIdInt, err := strconv.ParseInt(msgIdStr, 10, 64)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get message id", err.Error())
		return
	}
	err = api.QueueBounceMsg(msgIdInt)
	if err == gorm.ErrRecordNotFound {
		httpWriteErrorJson(w, 404, "no such message "+msgIdStr, "")
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to bounce message "+msgIdStr, err.Error())
		return
	}
}

// addQueueHandlers add Queue handlers to router
func addQueueHandlers(router *httprouter.Router) {
	// get all message in queue
	router.GET("/queue", wrapHandler(queueGetMessages))
	// get a message by id
	router.GET("/queue/:id", wrapHandler(queueGetMessage))
	// discard a message
	router.DELETE("/queue/discard/:id", wrapHandler(queueDiscardMessage))
	// bounce a message
	router.DELETE("/queue/bounce/:id", wrapHandler(queueBounceMessage))
}
