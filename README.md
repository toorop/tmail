# tmail

[![Join the chat at https://gitter.im/toorop/tmail](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/toorop/tmail?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

tmail is a SMTP server

## Features

 * SMTP, SMTP over SSL, ESMTP (SIZE, AUTH PLAIN, STARTTLS)
 * Advanced routing for outgoing mails (failover and round robin on routes, route by recipient, sender, authuser... )
 * SMTPAUTH (plain & cram-md5) for in/outgoing mails
 * STARTTLS/SSL for in/outgoing connexions.
 * Manageable via CLI or REST API.
 * DKIM support for signing outgoing mails.
 * Builtin support of clamav (open-source antivirus scanner).
 * Builtin Dovecot (imap server) support.
 * Fully extendable via plugins
 * Easy to deploy
 * No dependencies: -> you do not have to install nor maintain libs
 * Clusterisable (todo)
 * IPV6 (soon)


## Quick install on linux (Ubuntu)

For french users see: http://tmail.io/doc/installer-tmail/

### add user tmail

	adduser tmail

### Fetch tmail dist

	# su tmail
	$ cd
	$ wget ftp://ftp.toorop.fr/softs/tmail/tmail.zip
	$ unzip tmail.zip
	$ cd dist

Under dist you will find:

* conf: configuration.
* run: script used to launch tmail
* ssl: is the place to store SSL cert. For testing purpose you can use those included.
* tmail: tmail binary
* tpl: text templates.
* db: if you use sqlite as DB backend (MySQL and Postgresql are also supported), sqlite file will be strored in this directory.
* store: principaly used to store raw email when they are in queue. (others kind of backend/storage engine are comming)
* mailboxes: where mailboxes are stored if you activate Dovecot support.

Make run script and tmail runnable:

	chmod 700 run tmail

add directories:

	mkdir db
	mkdir store


if you want to enable Dovecot support add mailboxes directory:

	mkdir mailboxes

See [Enabling Dovecot support for tmail (french)](http://tmail.io/doc/mailboxes/) for more info.


### Configuration

Init you conf file:

	cd conf
	cp tmail.cfg.base tmail.cfg
	chmod 600 tmail.cfg

* TMAIL_ME: Hostname of the SMTP server (will be used for HELO|EHLO)

* TMAIL_DB_DRIVER: i recommend sqlite3 unless you want to enabled clustering (or you have a lot of domains/mailboxes)

* TMAIL_SMTPD_DSNS: listening IP(s), port(s) and SSL options (see conf file for more info)

* TMAIL_DELIVERD_LOCAL_IPS: IP(s) to use for sending mail to remote host.

* TMAIL_SMTPD_CONCURRENCY_INCOMING: max conccurent incomming proccess

* TMAIL_DELIVERD_MAX_IN_FLIGHT: concurrent delivery proccess


### Init database

	tmail@dev:~/dist$ ./run
	Database 'driver: sqlite3, source: /home/tmail/dist/db/tmail.db' misses some tables.
	Should i create them ? (y/n): y

	[dev.tmail.io - 127.0.0.1] 2015/02/02 12:42:32.449597 INFO - smtpd 151.80.115.83:2525 launched.
	[dev.tmail.io - 127.0.0.1] 2015/02/02 12:42:32.449931 INFO - smtpd 151.80.115.83:5877 launched.
	[dev.tmail.io - 127.0.0.1] 2015/02/02 12:42:32.450011 INFO - smtpd 151.80.115.83:4655 SSL launched.
	[dev.tmail.io - 127.0.0.1] 2015/02/02 12:42:32.499728 INFO - deliverd launched

### Port forwarding

As you run tmail under tmail user, it can't open port under 1024 (and for now tmail can be launched as root, open port under 25 and fork itself to unprivilegied user).

The workaround is to use iptables to forward ports.
For example, if we have tmail listening on ports 2525, and 5877 and we want tu use 25 and 587 as public ports, we have to use those iptables rules:

	iptables -t nat -A PREROUTING -p tcp --dport 25 -j REDIRECT --to-port 2525
	iptables -t nat -A PREROUTING -p tcp --dport 587 -j REDIRECT --to-port 5877

### First test

	$ telnet dev.tmail.io 25
	Trying 151.80.115.83...
	Connected to dev.tmail.io.
	Escape character is '^]'.
	220 tmail.io  tmail ESMTP f22815e0988b8766b6fe69cbc73fb0d965754f60
	HELO toto
	250 tmail.io
	MAIL FROM: toorop@tmail.io
	250 ok
	RCPT TO: toorop@tmail.io
	554 5.7.1 <toorop@tmail.io>: Relay access denied.
	Connection closed by foreign host.

Perfect !
You got "Relay access denied" because by default noboby can use tmail for relaying mails.

### Relaying mails for @example.com

If you want that tmail accepts to relay mails for example.com, just run:

	tmail rcpthost add example.com

Note: If you have activated Dovecot support and example.com is a local domain, add -l flag :

	tmail rcpthost add -l example.com

Does it work as exepected ?

	$ telnet dev.tmail.io 25
	Trying 151.80.115.83...
	Connected to dev.tmail.io.
	Escape character is '^]'.
	220 tmail.io  tmail ESMTP 96b78ef8f850253cc956820a874e8ce40773bfb7
	HELO toto
	250 tmail.io
	mail from: toorop@toorop.fr
	250 ok
	rcpt to: toorop@example.com
	250 ok
	data
	354 End data with <CR><LF>.<CR><LF>
	subject: test tmail

	blabla
	.
	250 2.0.0 Ok: queued 2736698d73c044fd7f1994e76814d737c702a25e
	quit
	221 2.0.0 Bye
	Connection closed by foreign host.

Yes ;)

### Allow relay from an IP

	tmail relayip add IP

For example:

	tmail relayip add 127.0.0.1


### Basic routing

By default tmail will use MX records for routing mails, but you can "manualy" configure alt routing.
Suppose that you want tmail to route mail fro @example.com to mx.slowmail.com. It as easy as:dd this routing rule

	tmail routes add -d example.com -rh mx.slowmail.com

You can find more elaborated routing rules on [tmail routing documentation (french)](http://tmail.io/doc/cli-gestion-route-smtp/) (translators are welcomed ;))

### SMTP AUTH

If you want to enable relaying after SMTP AUTH for user toorop@tmail.io, just enter:

	tmail user add -r toorop@tmail.io password


If you want to delete user toorop@tmail.io :

	tmail user del toorop@tmail.io


### Let's Encrypt (TLS/SSL)

If you want to activate TLS/SSL connexions with a valid certificate (not an auto-signed one as it's by default) between mail clients and your tmail server you can get a let's Encrypt certificate, you have first to install let's Encrypt :

	cd ~
	git clone https://github.com/letsencrypt/letsencrypt
	cd letsencrypt

Then you can request a certificate

	./letsencrypt-auto certonly --standalone -d your.hostname

You'll have to provide a valid mail address and agree to the Let's Encrypt Term of Service. When certificate is issued you have to copy some files to the ssl/ directory

	cd /home/tmail/dist/ssl
	cp /etc/letsencrypt/live/your.hostname/fullchain.pem server.crt
	cp /etc/letsencrypt/live/your.hostname/privkey.pem server.key
	chown tmail.tmail server.*

And it's done !


## Contribute

Feel free to inspect & improve tmail code, PR are welcomed ;)

If you are not a coder, you can contribute too:

* install and use tmail, i need feebacks.

* as you can see reading this page, english is not my native language, so i need help to write english documentation.


## Roadmap

 * clustering
 * IPV6
 * write unit tests (yes i know...)
 * improve, refactor, optimize
 * test test test test


## License
MIT, see LICENSE


## Imported packages

github.com/nsqio/nsq/...
github.com/codegangsta/cli
github.com/codegangsta/negroni
github.com/go-sql-driver/mysql
github.com/jinzhu/gorm
github.com/julienschmidt/httprouter
github.com/kless/osutil/user/crypt/...
github.com/lib/pq
github.com/mattn/go-sqlite3
github.com/nbio/httpcontext
golang.org/x/crypto/bcrypt
golang.org/x/crypto/blowfish
