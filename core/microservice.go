package core

import (
	// "errors"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/toorop/tmail/msproto"
	"github.com/toorop/tmail/scope"
)

type onfailure int

const (
	CONTINUE onfailure = 1 + iota
	TEMPFAIL
	PERMFAIL
)

// microservice represents a microservice
type microservice struct {
	url           string
	fireAndForget bool
	timeout       uint64
	onFailure     onfailure
}

// newMicroservice retuns a microservice parsing URI
func newMicroservice(uri string) (*microservice, error) {
	ms := &microservice{
		onFailure: CONTINUE,
		timeout:   30,
	}
	t := strings.Split(uri, "?")
	ms.url = t[0]
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if parsed.Query().Get("fireandforget") == "true" {
		ms.fireAndForget = true
	}
	if parsed.Query().Get("timeout") != "" {
		ms.timeout, err = strconv.ParseUint(parsed.Query().Get("timeout"), 10, 64)
		if err != nil {
			return nil, err
		}
	}
	if parsed.Query().Get("onfailure") != "" {
		switch parsed.Query().Get("onfailure") {
		case "tempfail":
			ms.onFailure = TEMPFAIL
		case "permfail":
			ms.onFailure = PERMFAIL
		}
	}
	return ms, nil
}

// doRequest do request on microservices endpoint
func (ms *microservice) doRequest(data *[]byte) (*http.Response, error) {
	req, _ := http.NewRequest("POST", ms.url, bytes.NewBuffer(*data))
	req.Header.Set("Content-Type", "application/x-protobuf")
	client := &http.Client{
		Timeout: time.Duration(ms.timeout) * time.Second,
	}
	return client.Do(req)

}

// smtpdExec exec microservice
func (ms *microservice) smtpdExec(data *[]byte) (*msproto.SmtpdResponse, error) {

	// HTTP resquest
	r, err := ms.doRequest(data)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// HTTP code > 399
	if r.StatusCode > 399 {
		return nil, errors.New(string(body))
	}

	// parse data as Smtpdresponse
	resp := &msproto.SmtpdResponse{}
	err = proto.Unmarshal(body, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// smtpdBreakOnExecError handle error when calling a ms
// it returns true if tmail must stop processing other ms
func (ms *microservice) smtpdBreakOnExecError(err error, s *smtpServerSession) (stop bool) {
	if err == nil {
		return false
	}
	s.logError("ms " + ms.url + " failed. " + err.Error())
	switch ms.onFailure {
	case PERMFAIL:
		s.out("550 sorry something wrong happened")
		s.exitAsap()
		return true
	case TEMPFAIL:
		s.exitAsap()
		s.out("450 sorry something wrong happened")
		return true
	default:
		return false
	}
}

// smtpdHandleResponse common handling of msproto.SmtpdResponse
func smtpdHandleResponse(resp *msproto.SmtpdResponse, s *smtpServerSession) (stop, sendDefaultReply bool) {
	s.logDebug(resp.String())
	if resp.GetSmtpCode() != 0 && resp.GetSmtpMsg() != "" {
		outMsg := fmt.Sprintf("%d %s", resp.GetSmtpCode(), resp.GetSmtpMsg())
		s.log("ms smtp response: " + outMsg)
		s.out(outMsg)
		sendDefaultReply = false
		if resp.GetCloseConnection() {
			s.exitAsap()
			return true, false
		}
		return false, false
	}
	return false, true
}

// smtpdNewClient execute microservices for smtpdnewclient hook
func smtpdNewClient(s *smtpServerSession) (stop, sendDefaultReply bool) {
	stop = false
	sendDefaultReply = true
	for _, uri := range scope.Cfg.GetMicroservicesUri("smtpdnewclient") {
		s.log("call microservice: " + uri)
		// parse uri
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to parse microservice url " + uri + ". " + err.Error())
			continue
		}

		// serialize message to send
		data, err := proto.Marshal(&msproto.SmtpdNewClientMsg{
			SessionId: proto.String(s.uuid),
			RemoteIp:  proto.String(s.conn.RemoteAddr().String()),
		})
		if err != nil {
			s.logError("unable to serialize ms data as SmtpdNewClientMsg. " + err.Error())
			continue
		}

		// call ms
		if ms.fireAndForget {
			go ms.smtpdExec(&data)
			continue
		}

		resp, err := ms.smtpdExec(&data)

		// Handle error from MS
		if ms.smtpdBreakOnExecError(err, s) {
			return true, false
		}

		// Handle response
		stop, sendDefaultReply = smtpdHandleResponse(resp, s)
		if stop {
			return stop, sendDefaultReply
		}
	}
	return
}

// smtpdFunc map of function corresponding to a hook
var smtpdFunc = map[string]func(s *smtpServerSession) (stop, sendDefaultReply bool){
	"smtpdNewClient": smtpdNewClient,
}

// msSmtptdCall call a microservice fro smtpd session
func msSmtptdCall(hookId string, session *smtpServerSession) (stop, sendDefaultReply bool) {
	if fn, ok := smtpdFunc[hookId]; ok {
		return fn(session)
	}
	return false, true
}
