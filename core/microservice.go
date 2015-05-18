package core

import (
	// "errors"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

func (ms *microservice) smtpdExec(data *[]byte) (*msproto.SmtpdResponse, error) {
	req, _ := http.NewRequest("POST", ms.url, bytes.NewBuffer(*data))
	req.Header.Set("Content-Type", "application/x-protobuf")
	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	*data, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// parse data
	resp := &msproto.SmtpdResponse{}
	err = proto.Unmarshal(*data, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (ms *microservice) smtpdBreakOnExecError(err error, s *smtpServerSession) (stop bool) {
	if err == nil {
		return false
	}
	s.logError("ms " + ms.url + " failed. " + err.Error())
	switch ms.onFailure {
	case PERMFAIL:
		s.out("550 sorry something wrong happened")
		return true
	case TEMPFAIL:
		s.out("450 sorry something wrong happened")
		return true
	default:
		return false
	}

}

func smtpdNewClient(s *smtpServerSession) bool {
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
		resp, err := ms.smtpdExec(&data)
		if ms.smtpdBreakOnExecError(err, s) {
			return false
		}

		fmt.Println(resp.GetSmtpCode())

	}
	return false
}

var smtpdFunc = map[string]func(s *smtpServerSession) bool{
	"smtpdNewClient": smtpdNewClient,
}

// msSmtptdCall call a microservice fro smtpd session
func msSmtptdCall(hookId string, session *smtpServerSession) bool {
	if fn, ok := smtpdFunc[hookId]; ok {
		return fn(session)
	}
	/*if smtpdFunc[hookId] {
		return smtpdFunc[hookId](session)
	}*/
	// log errors.New("unavalaible hookId " + hookId)
	return false
}
