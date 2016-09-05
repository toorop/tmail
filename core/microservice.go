package core

import (
	// "errors"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/toorop/tmail/msproto"
)

type onfailure int

// what to do on failure
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

// exec a microservice
func (ms *microservice) exec(s *SMTPServerSession, queryMsg *[]byte) (response *[]byte, err error) {
	if s.user != nil && ms.skipAuthentifiedUser {
		return
	}
	// call ms
	s.log("calling " + ms.url)
	if ms.fireAndForget {
		go ms.call(queryMsg)
		return
	}
	response, err = ms.call(queryMsg)
	return
}

// shouldWeStopOnError return if process wich call microserviuce should stop on error
func (ms *microservice) stopOnError() (stop bool) {
	switch ms.onFailure {
	case PERMFAIL:
		return true
	case TEMPFAIL:
		return true
	default:
		return false
	}
}

// smtpdStopOnError handle error for smtpd microservice
// it returns true if tmail must stop processing other ms
// handleSMTPError
func (ms *microservice) handleSMTPError(err error, s *SMTPServerSession) (stop bool) {
	if err == nil {
		return false
	}
	s.logError("microservice " + ms.url + " failed. " + err.Error())
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

// handleSmtpResponse common handling of msproto.SmtpdResponse
func handleSMTPResponse(smtpResponse *msproto.SmtpResponse, s *SMTPServerSession) (stop bool) {
	if smtpResponse == nil {
		return
	}
	if smtpResponse.GetCode() != 0 && smtpResponse.GetMsg() != "" {
		reply := fmt.Sprintf("%d %s", smtpResponse.GetCode(), smtpResponse.GetMsg())
		s.out(reply)
		s.log("smtp response from microservice sent to client: " + reply)
		// if reply is sent we do not continue processing this command
		stop = true
	}
	return
}

// msSmtpdNewClient execute microservices for smtpdnewclient hook
// Warning: error are not returned to client
func msSmtpdNewClient(s *SMTPServerSession) (stop bool) {
	if len(Cfg.GetMicroservicesUri("smtpdnewclient")) == 0 {
		return false
	}

	// serialize message to send
	msg, err := proto.Marshal(&msproto.SmtpdNewClientQuery{
		SessionId: proto.String(s.uuid),
		RemoteIp:  proto.String(s.conn.RemoteAddr().String()),
	})
	if err != nil {
		s.logError("unable to serialize data as SmtpdNewClientMsg. " + err.Error())
		return
	}

	for _, uri := range Cfg.GetMicroservicesUri("smtpdnewclient") {
		stop = false
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to parse microservice url " + uri + ". " + err.Error())
			continue
		}

		response, err := ms.exec(s, &msg)
		if err != nil {
			s.logError("microservice " + ms.url + " failed. " + err.Error())
			if ms.stopOnError() {
				return
			}
		}

		// parse resp
		msResponse := &msproto.SmtpdNewClientResponse{}
		if err = proto.Unmarshal(*response, msResponse); err != nil {
			s.logError("microservice " + ms.url + " failed. " + err.Error())
			if ms.stopOnError() {
				return
			}
			continue
		}

		// send reply (or not)
		stop = handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		// drop ?
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true
		}
	}
	return
}

// msSmtpdHelo microservice called after HELO/EHLO cmd recieved
func msSmtpdHelo(s *SMTPServerSession, helo []string) (stop bool) {
	var response *[]byte
	var msResponse *msproto.SmtpdHeloResponse
	var ms *microservice
	var err error

	// get URIs
	uris := Cfg.GetMicroservicesUri("smtpdhelo")
	if len(uris) == 0 {
		return
	}
	// Query
	msg, err := proto.Marshal(&msproto.SmtpdHeloQuery{
		SessionId: proto.String(s.uuid),
		Helo:      proto.String(strings.Join(helo, " ")),
	})
	if err != nil {
		s.logError("ms - unable marshall SmtpdHeloQuery " + err.Error())
		return
	}

	for _, uri := range uris {
		ms, err = newMicroservice(uri)
		if err != nil {
			s.logError("ms - unable to init microservice msSmtpdHelo -" + err.Error())
			continue
		}
		if response, err = ms.exec(s, &msg); err != nil {
			s.logError("ms - unable to call microservice msSmtpdHelo -" + err.Error())
			continue
		}
		// unmarshal response
		msResponse = &msproto.SmtpdHeloResponse{}
		if err = proto.Unmarshal(*response, msResponse); err != nil {
			s.logError("microservice " + ms.url + " failed. " + err.Error())
			if ms.stopOnError() {
				return
			}
			continue
		}

		// send reply (or not)
		stop = handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		// drop ?
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true
		}
	}
	return
}

// msSmtpdMailFrom microservice called after MAIL FROM SMTP command
func msSmtpdMailFrom(s *SMTPServerSession, mailFrom []string) (stop bool) {
	var response *[]byte
	var msResponse *msproto.SmtpdMailFromResponse
	var ms *microservice
	var err error

	// get URIs
	uris := Cfg.GetMicroservicesUri("smtpdmailfrom")
	if len(uris) == 0 {
		return
	}
	// Query
	msg, err := proto.Marshal(&msproto.SmtpdMailFromQuery{
		SessionId: proto.String(s.uuid),
		From:      proto.String(strings.Join(mailFrom, " ")),
	})
	if err != nil {
		s.logError("ms - unable marshall SmtpdMailFromQuery " + err.Error())
		return
	}

	for _, uri := range uris {
		ms, err = newMicroservice(uri)
		if err != nil {
			s.logError("ms - unable to init microservice SmtpdMailFromQuery -" + err.Error())
			continue
		}
		if response, err = ms.exec(s, &msg); err != nil {
			s.logError("ms - unable to call microservice SmtpdMailFromQuery -" + err.Error())
			continue
		}
		// unmarshal response
		msResponse = &msproto.SmtpdMailFromResponse{}
		if err = proto.Unmarshal(*response, msResponse); err != nil {
			s.logError("microservice " + ms.url + " failed. " + err.Error())
			if ms.stopOnError() {
				return
			}
			continue
		}

		// send reply (or not)
		stop = handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		// drop ?
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true
		}
	}
	return
}

// msSmtpdRcptTo check if relay is granted by using rcpt to
func msSmtpdRcptTo(s *SMTPServerSession, rcptTo string) (stop bool) {
	if len(Cfg.GetMicroservicesUri("smtpdrcptto")) == 0 {
		return false
	}
	msg, err := proto.Marshal(&msproto.SmtpdRcptToQuery{
		SessionId: proto.String(s.uuid),
		MailFrom:  proto.String(s.envelope.MailFrom),
		RcptTo:    proto.String(rcptTo),
	})
	if err != nil {
		s.logError("unable to serialize data as SmtpdRcptToQuery. " + err.Error())
		return
	}

	for _, uri := range Cfg.GetMicroservicesUri("smtpdrcptto") {
		stop = false
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to parse microservice url " + uri + ". " + err.Error())
			continue
		}

		if s.user != nil && ms.skipAuthentifiedUser {
			continue
		}

		// call ms
		s.log("calling " + ms.url)
		if ms.fireAndForget {
			go ms.call(&msg)
			continue
		}

		response, err := ms.call(&msg)
		if err != nil {
			if stop := ms.handleSMTPError(err, s); stop {
				return true
			}
			continue
		}

		// parse resp
		msResponse := &msproto.SmtpdRcptToResponse{}
		err = proto.Unmarshal(*response, msResponse)
		if err != nil {
			if stop := ms.handleSMTPError(err, s); stop {
				return true
			}
			continue
		}

		// Relay granted
		s.relayGranted = msResponse.GetRelayGranted()

		// send reply (or not)
		stop = handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		// drop ?
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true
		}

	}
	return stop
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
	msg, err := proto.Marshal(&msproto.SmtpdDataQuery{
		SessionId: proto.String(s.uuid),
		DataLink:  proto.String(link),
		Enveloppe: proto.String(s.envelope.String()),
	})
	if err != nil {
		s.logError("unable to serialize data as SmtpdDataQuery. " + err.Error())
		return
	}

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

		s.log("calling " + ms.url)
		response, err := ms.call(&msg)
		if err != nil {
			if stop := ms.handleSMTPError(err, s); stop {
				return true, extraHeaders
			}
			continue
		}

		// parse resp
		msResponse := &msproto.SmtpdDataResponse{}
		err = proto.Unmarshal(*response, msResponse)
		if err != nil {
			if stop := ms.handleSMTPError(err, s); stop {
				return true, extraHeaders
			}
			continue
		}

		*extraHeaders = append(*extraHeaders, msResponse.GetExtraHeaders()...)

		// send reply (or not)
		stop = handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		// drop ?
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true, extraHeaders
		}
	}
	return false, extraHeaders
}

func msSmtpdSendTelemetry(s *SMTPServerSession) {
	msURI := Cfg.GetMicroservicesUri("smtpdsendtelemetry")
	if len(msURI) == 0 {
		return
	}
	telemetry := msproto.SmtpdTelemetry{}
	telemetry.ServerId = proto.String(Cfg.GetMe())
	telemetry.SessionId = proto.String(s.uuid)
	telemetry.RemoteAddress = proto.String(s.remoteAddr)
	telemetry.EnvMailfrom = proto.String(s.envelope.MailFrom)
	telemetry.EnvRcptto = s.envelope.RcptTo
	if s.SMTPResponseCode != 0 && s.SMTPResponseCode < 400 {
		telemetry.Success = proto.Bool(true)
	} else {
		telemetry.Success = proto.Bool(false)
	}
	telemetry.MessageSize = proto.Uint32(s.dataBytes)
	telemetry.IsTls = proto.Bool(s.tls)
	telemetry.Concurrency = proto.Uint32(uint32(SmtpSessionsCount))
	telemetry.SmtpResponseCode = proto.Uint32(s.SMTPResponseCode)
	telemetry.ExecTime = proto.Uint32(uint32(time.Since(s.startAt).Nanoseconds()))

	// metrics are collected we release s
	go func(sessionID string) {
		msg, err := proto.Marshal(&telemetry)
		if err != nil {
			Log.Error(fmt.Sprintf("smtpd - %s - msSmtpdSendTelemetry - unable to serialize message: %s", sessionID, err.Error()))
			return
		}
		for _, uri := range Cfg.GetMicroservicesUri("smtpdsendtelemetry") {
			ms, err := newMicroservice(uri)
			if err != nil {
				Log.Error(fmt.Sprintf("smtpd - %s - msSmtpdSendTelemetry - unable to init new ms: %s", sessionID, err.Error()))
				continue
			}
			Log.Info(fmt.Sprintf("smtpd - %s - msSmtpdSendTelemetry - call ms: %s", sessionID, ms.url))
			if _, err = ms.call(&msg); err != nil {
				Log.Error(fmt.Sprintf("smtpd - %s - msSmtpdSendTelemetry - unable to call ms: %s", sessionID, err.Error()))
				continue
			}
		}
	}(s.uuid)
	return
}

// msSmtpdBeforeQueueing -> edit enveloppe.
// return stop
func msSmtpdBeforeQueueing(s *SMTPServerSession) bool {
	msURI := Cfg.GetMicroservicesUri("smtpdbeforequeueing")
	if len(msURI) == 0 {
		return false
	}

	for _, uri := range Cfg.GetMicroservicesUri("smtpdbeforequeueing") {
		// parse uri
		ms, err := newMicroservice(uri)
		if err != nil {
			s.logError("unable to get microservice " + uri)
			if ms.handleSMTPError(err, s) {
				return true
			}
			continue
		}
		// Query -> warning order is important
		query, err := proto.Marshal(&msproto.SmtpdBeforeQueueingQuery{
			SessionId: proto.String(s.uuid),
			MailFrom:  proto.String(s.envelope.MailFrom),
			RcptTo:    s.envelope.RcptTo,
		})
		if err != nil {
			s.logError("unable to serialize ms message for msSmtpdBeforeQueueing - ", err.Error())
			if ms.handleSMTPError(err, s) {
				return true
			}
			continue
		}

		response, err := ms.call(&query)
		if err != nil {
			s.logError("msSmtpdBeforeQueueing failed on call " + err.Error())
			if ms.handleSMTPError(err, s) {
				return true
			}
			continue
		}
		// parse response
		msResponse := &msproto.SmtpdBeforeQueueingResponse{}
		err = proto.Unmarshal(*response, msResponse)
		if err != nil {
			s.logError("unable to unmarshal response from ms SmtpdBeforeQueueingResponse")
			if ms.handleSMTPError(err, s) {
				return true
			}
			continue
		}
		// New mail from ?
		if msResponse.GetMailFrom() != "" {
			if address, err := mail.ParseAddress(msResponse.GetMailFrom()); err == nil {
				s.envelope.MailFrom = address.Address
			}
		}

		// New rcpt to ?
		if len(msResponse.GetRcptTo()) != 0 {
			newRcptTo := []string{}
			// check validity of email
			for _, rawAddress := range msResponse.GetRcptTo() {
				address, err := mail.ParseAddress(rawAddress)
				if err == nil {
					newRcptTo = append(newRcptTo, address.Address)
				}

			}
			if len(newRcptTo) != 0 {
				s.envelope.RcptTo = newRcptTo
			}

		}
		stop := handleSMTPResponse(msResponse.GetSmtpResponse(), s)
		if msResponse.GetDropConnection() {
			s.exitAsap()
			stop = true
		}
		if stop {
			return true
		}
	}
	return false

}

// msGetRoutesmsGetRoutes returns routes from microservices
func msGetRoutes(d *delivery) (routes *[]Route, stop bool) {
	stop = false
	r := []Route{}
	routes = &r
	msURI := Cfg.GetMicroservicesUri("deliverdgetroutes")
	if len(msURI) == 0 {
		return
	}

	// There should be only one URI for getroutes
	// so we take msURI[0]
	ms, err := newMicroservice(msURI[0])
	if err != nil {
		//Log.Error("deliverd-ms " + d.id + ": unable to parse microservice url " + msURI[0] + " - " + err.Error())
		Log.Error(fmt.Sprintf("deliverd-remote %s - msGetRoutes - unable to init new ms: %s", d.id, err.Error()))
		return nil, ms.stopOnError()
	}

	// serialize data
	msg, err := proto.Marshal(&msproto.DeliverdGetRoutesQuery{
		DeliverdId:       proto.String(d.id),
		Mailfrom:         proto.String(d.qMsg.MailFrom),
		Rcptto:           proto.String(d.qMsg.RcptTo),
		AuthentifiedUser: proto.String(d.qMsg.AuthUser),
	})
	if err != nil {
		//Log.Error("deliverd-ms " + d.id + ": unable to parse microservice url " + msURI[0] + " - " + err.Error())
		Log.Error(fmt.Sprintf("deliverd-remote %s - msGetRoutes - unable to serialize new ms: %s", d.id, err.Error()))
		return nil, ms.stopOnError()
	}

	Log.Info(fmt.Sprintf("deliverd-remote %s - msGetRoutes - call ms: %s", d.id, ms.url))
	response, err := ms.call(&msg)
	if err != nil {
		Log.Error(fmt.Sprintf("deliverd-remote %s - msGetRoutes - unable to call ms: %s", d.id, err.Error()))
		return nil, ms.stopOnError()
	}

	// parse resp
	msResponse := &msproto.DeliverdGetRoutesResponse{}
	if err := proto.Unmarshal(*response, msResponse); err != nil {
		Log.Error(fmt.Sprintf("deliverd-remote %s - msGetRoutes - unable to unmarshall response: %s", d.id, err.Error()))
		return routes, ms.stopOnError()
	}
	// no routes found
	if len(msResponse.GetRoutes()) == 0 {
		return nil, false
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
	return routes, false
}

// msDeliverdSendTelemetry send deliverd telemetry
func msDeliverdSendTelemetry(d *delivery) {
	msURI := Cfg.GetMicroservicesUri("deliverdsendtelemetry")
	if len(msURI) == 0 {
		return
	}

	telemetry := msproto.DeliverdTelemetry{}
	telemetry.ServerId = proto.String(Cfg.GetMe())
	telemetry.DeliverdId = proto.String(d.id)
	telemetry.Success = proto.Bool(d.success)
	telemetry.ExecTime = proto.Uint32(uint32(time.Since(d.startAt).Nanoseconds()))
	t, err := QueueCount()
	if err != nil {
		Log.Error(fmt.Sprintf("deliverd-remote %s - msDeliverdSendTelemetry - unable to get QueueCount %s", d.id, err.Error()))
		return
	}
	telemetry.MessagesInQueue = proto.Uint32(t)
	telemetry.ConcurrencyRemote = proto.Uint32(uint32(DeliverdConcurrencyRemoteCount))
	telemetry.ConcurrencyLocal = proto.Uint32(uint32(DeliverdConcurrencyLocalCount))
	telemetry.IsLocal = proto.Bool(d.isLocal)
	telemetry.From = proto.String(d.qMsg.MailFrom)
	telemetry.To = proto.String(d.qMsg.RcptTo)
	if !d.isLocal {
		telemetry.RemoteAddress = proto.String(d.remoteAddr)
		telemetry.LocalAddress = proto.String(d.localAddr)
		telemetry.RemoteSmtpResponseCode = proto.Uint32(uint32(d.remoteSMTPresponseCode))
	}
	//
	go func(deliveryID string) {
		msg, err := proto.Marshal(&telemetry)
		if err != nil {
			Log.Error(fmt.Sprintf("deliverd-remote %s - msDeliverdSendTelemetry - unable to serialize message: %s", deliveryID, err.Error()))
			return
		}
		for _, uri := range Cfg.GetMicroservicesUri("deliverdsendtelemetry") {
			ms, err := newMicroservice(uri)
			if err != nil {
				Log.Error(fmt.Sprintf("deliverd-remote %s - msDeliverdSendTelemetry - unable to init new ms: %s", deliveryID, err.Error()))
				continue
			}
			Log.Info(fmt.Sprintf("deliverd-remote %s - msDeliverdSendTelemetry - call ms: %s", deliveryID, ms.url))
			if _, err = ms.call(&msg); err != nil {
				Log.Error(fmt.Sprintf("deliverd-remote %s - msDeliverdSendTelemetry - unable to call ms: %s", deliveryID, err.Error()))
				continue
			}
		}
	}(d.id)
	return
}
