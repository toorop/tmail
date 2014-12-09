Date: {{.Date}}
From: MAILER-DAEMON@{{.Me}}
To: {{.RcptTo}}
Subject: failure notice

Hi. This is the tmail deliverd program at {{.Me}}
I'm afraid I wasn't able to deliver your message to the
following addresses. This is a permanent error; I've given up.
Sorry it didn't work out.

<{{.RcptTo}}>:
{{.ErrMsg}}

--- Below this line is a copy of the message.

{{.BouncedMail}}