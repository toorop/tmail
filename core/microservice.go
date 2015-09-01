package core

import (
	// "errors"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/toorop/tmail/msproto"
)

type onfailure int

const (
	CONTINUE onfailure = 1 + iota
	TEMPFAIL
	PERMFAIL
)

// microservice represents a microservice
type microservice struct {
	url                  string
	skipAuthentifiedUser bool
	fireAndForget        bool
	timeout              uint64
	onFailure            onfailure
}

// newMicroservice retuns a microservice parsing URI
func newMicroservice(uri string) (*microservice, error) {
	ms := &microservice{
		skipAuthentifiedUser: false,
		onFailure:            CONTINUE,
		timeout:              30,
	}
	t := strings.Split(uri, "?")
	ms.url = t[0]
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if parsed.Query().Get("skipauthentifieduser") == "true" {
		ms.skipAuthentifiedUser = true
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

// call will call microservice
func (ms *microservice) call(data *[]byte) (*[]byte, error) {
	r, err := ms.doRequest(data)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	// always get returned data
	rawBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// HTTP error handling
	if r.StatusCode > 399 {
		return nil, errors.New(r.Status + " - " + string(rawBody))
	}
	return &rawBody, nil
}

// smtpdExec exec microservice
func (ms *microservice) smtpdExec(data *[]byte) (*msproto.SmtpdResponse, error) {
	response, err := ms.call(data)
	if err != nil {
		return nil, err
	}
	// parse data as Smtpdresponse
	msResponse := &msproto.SmtpdResponse{}
	err = proto.Unmarshal(*response, msResponse)
	if err != nil {
		return nil, err
	}
	return msResponse, nil
}

// smtpdStopOnError handle error for smtpd microservice
// it returns true if tmail must stop processing other ms
func (ms *microservice) smtpdStopOnError(err error, s *SMTPServerSession) (stop bool) {
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
func smtpdReturn(resp *msproto.SmtpdResponse, s *SMTPServerSession) (stop bool) {
	s.logDebug(resp.String())
	if resp.GetSmtpCode() != 0 && resp.GetSmtpMsg() != "" {
		outMsg := fmt.Sprintf("%d %s", resp.GetSmtpCode(), resp.GetSmtpMsg())
		s.log("ms smtp response: " + outMsg)
		s.out(outMsg)
		if resp.GetCloseConnection() {
			s.exitAsap()
		}
		return true
	}
	return false
}

// msSmtpdNewClient execute microservices for smtpdnewclient hook
func msSmtpdNewClient(s *SMTPServerSession) (stop bool) {
	if len(Cfg.GetMicroservicesUri("smtpdnewclient")) == 0 {
		return false
	}
	stop = false

	// serialize message to send
	data, err := proto.Marshal(&msproto.SmtpdNewClientMsg{
		SessionId: proto.String(s.uuid),
		RemoteIp:  proto.String(s.conn.RemoteAddr().String()),
	})
	if err != nil {
		s.logError("unable to serialize ms data as SmtpdNewClientMsg. " + err.Error())
		return
	}

	for _, uri := range Cfg.GetMicroservicesUri("smtpdnewclient") {
		// parse uri
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to parse microservice url " + uri + ". " + err.Error())
			continue
		}

		if s.user != nil && ms.skipAuthentifiedUser {
			continue
		}

		// call ms
		s.log("call ms " + uri)
		if ms.fireAndForget {
			go ms.smtpdExec(&data)
			continue
		}

		resp, err := ms.smtpdExec(&data)

		// Handle error from MS
		if ms.smtpdStopOnError(err, s) {
			return true
		}

		// Handle response
		if smtpdReturn(resp, s) {
			return true
		}
	}
	return
}

// msSmtpdRcptToRelayIsGranted check if relay is granted by using rcpt to
func msSmtpdRcptToRelayIsGranted(s *SMTPServerSession, rcpt string) (stop bool) {
	msURI := Cfg.GetMicroservicesUri("smtpdrcptorelayisgranted")
	if len(msURI) == 0 {
		return false
	}

	ms, err := newMicroservice(msURI[0])
	if err != nil {
		return ms.smtpdStopOnError(err, s)
	}

	s.log(s.uuid, rcpt)

	msg, err := proto.Marshal(&msproto.SmtpdRcpttoAccessIsGrantedQuery{
		SessionId: proto.String(s.uuid),
		Rcptto:    proto.String(rcpt),
	})
	if err != nil {
		return ms.smtpdStopOnError(err, s)
	}

	response, err := ms.call(&msg)
	if err != nil {
		return ms.smtpdStopOnError(err, s)
	}

	// parse resp
	msResponse := &msproto.SmtpdRcpttoAccessIsGrantedResponse{}
	err = proto.Unmarshal(*response, msResponse)
	if err != nil {
		return ms.smtpdStopOnError(err, s)
	}
	s.relayGranted = msResponse.GetRelayGranted()
	return false
}

// smtpdData executes microservices for the smtpdData hook
func smtpdData(s *SMTPServerSession, rawMail *[]byte) (stop bool, extraHeaders *[]string) {
	extraHeaders = &[]string{}

	if len(Cfg.GetMicroservicesUri("smtpddata")) == 0 {
		return false, extraHeaders
	}

	// save data to server throught HTTP
	f, err := ioutil.TempFile(Cfg.GetTempDir(), "")
	if err != nil {
		s.logError("ms - unable to save rawmail in tempfile. " + err.Error())
		return false, extraHeaders
	}
	if _, err = f.Write(*rawMail); err != nil {
		s.logError("ms - unable to save rawmail in tempfile. " + err.Error())
		return false, extraHeaders
	}
	defer os.Remove(f.Name())

	// HTTP link
	t := strings.Split(f.Name(), "/")
	link := fmt.Sprintf("%s:%d/msdata/%s", Cfg.GetRestServerIp(), Cfg.GetRestServerPort(), t[len(t)-1])

	// TLS
	if Cfg.GetRestServerIsTls() {
		link = "https://" + link
	} else {
		link = "http://" + link
	}

	// serialize data
	msg, err := proto.Marshal(&msproto.SmtpdDataMsg{
		SessionId: proto.String(s.uuid),
		DataLink:  proto.String(link),
	})

	for _, uri := range Cfg.GetMicroservicesUri("smtpddata") {
		// parse uri
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to parse microservice url " + uri + ". " + err.Error())
			continue
		}
		if s.user != nil && ms.skipAuthentifiedUser {
			continue
		}
		s.log("call ms " + uri)

		resp, err := ms.smtpdExec(&msg)
		// Handle error from MS
		if ms.smtpdStopOnError(err, s) {
			return true, nil
		}
		*extraHeaders = append(*extraHeaders, resp.GetExtraHeaders()...)
		if smtpdReturn(resp, s) {
			return true, extraHeaders
		}
	}
	return false, extraHeaders
}

// msGetRoutesmsGetRoutes returns routes from microservices
func msGetRoutes(d *delivery) (routes *[]Route, err error) {
	r := []Route{}
	routes = &r
	msURI := Cfg.GetMicroservicesUri("deliverdgetroutes")
	if len(msURI) == 0 {
		return
	}
	// serialize data
	msg, err := proto.Marshal(&msproto.DeliverdGetRoutesQuery{
		DeliverdId:       proto.String(d.id),
		Mailfrom:         proto.String(d.qMsg.MailFrom),
		Rcptto:           proto.String(d.qMsg.RcptTo),
		AuthentifiedUser: proto.String(d.qMsg.AuthUser),
	})

	// There should be only one URI for getroutes
	// so we take msURI[0]
	ms, err := newMicroservice(msURI[0])
	if err != nil {
		Log.Error("deliverd-ms " + d.id + ": unable to parse microservice url " + msURI[0] + " - " + err.Error())
		return
	}

	response, err := ms.call(&msg)
	if err != nil {
		return nil, err
	}

	// parse resp
	msResponse := &msproto.DeliverdGetRoutesResponse{}
	if err := proto.Unmarshal(*response, msResponse); err != nil {
		return nil, err
	}
	// no routes found
	if len(*routes) == 0 {
		return nil, nil
	}
	for _, route := range msResponse.GetRoutes() {
		r := Route{
			RemoteHost: route.GetRemoteHost(),
		}
		if route.GetLocalIp() != "" {
			r.LocalIp = sql.NullString{String: route.GetLocalIp(), Valid: true}
		}
		if route.GetRemotePort() != 0 {
			r.RemotePort = sql.NullInt64{Int64: int64(route.GetRemotePort()), Valid: true}
		}
		if route.GetPriority() != 0 {
			r.Priority = sql.NullInt64{Int64: int64(route.GetPriority()), Valid: true}
		}
		*routes = append(*routes, r)
	}
	return
}

/*
// smtpdFunc map of function corresponding to a hook
var smtpdFunc = map[string]func(i ...interface{}) (stop, sendDefaultReply bool){
	"smtpdNewClient": smtpdNewClient,
}

// msSmtptdCall call a microservice fro smtpd session
func msSmtptdCall(hookId string, i ...interface{}) (stop, sendDefaultReply bool) {
	if fn, ok := smtpdFunc[hookId]; ok {
		return fn(i)
	}
	return false, true
}
*/
