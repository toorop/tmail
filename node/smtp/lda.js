#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 *
 * This fiel act has default delivery agent for qmail
 *
 * echo "|/var/qmail/bin/preline -f /home/tmail/node/smtp/lda.js -d $EXT@$USER" > /var/qmail/control/defaultdelivery
 *
 */

var uuid = require('node-uuid');
//var querystring = require("querystring");
var crypto = require('crypto');

var mailSize=0; // Size of the mail in Octet

/* Riak */
var riak = require("riak-js").getClient({host: "10.10.0.1", port: "8098", debug: true });

var MailParser = require("mailparser").MailParser,
    mailparser = new MailParser({debug: false, showAttachmentLinks: true, streamAttachments: true}),
    fs = require("fs");

mailparser.on("attachment", function (attachment) {
    //console.log(attachment);
    //var output = fs.createWriteStream('/var/www/protecmail.com/Symfony/web/quarantine/' + messageID + '_' + attachment.generatedFileName);
    //attachment.stream.pipe(output);
});

mailparser.on("end", function (mail) {

    // envelope sender
    mail.from = process.env.SENDER;

    // envelope From
    mail.to = process.env.RECIPIENT;
    if (!mail.to) {
        mail.to = mail.headers.to;
    }

    // Subject
    mail.subject = mail.headers['subject'];

    // Size
    mail.size=mailSize;

    // date
    if (mail.headers['dated']) {
        mail.date = Date.parse(mail.headers['date']);
    } else mail.date=new Date().getTime();

    mail.msgId = mail.headers['message-id'];
    if (mail.msgId) { // remove <>
        mail.msgId = mail.msgId.substr(1,mail.msgId.length - 2);
    } else {
        mail.msgId = uuid.v1();
    }
    //msgId = querystring.escape(msgId);

    // @todo : il faut tester les filtres

    // les labels
    mail.labels = [];
    // inbox
    mail.labels.push('inbox');

    // seen
    mail.seen = false;

    // threadId
    if (!mail.threadIndex) {
        if (mail.references) {
            mail.threadIndex = mail.references[0];
        } else mail.threadIndex = "msgId";
    }

    // prev & next
    // @todo Riak link
    if (mail.inReplyTo) {
        mail.prev = mail.inReplyTo;
        delete mail.inReplyTo;
    } else mail.prev = "";
    mail.next = "";

    // on doit nettoyer les attachements
    if (mail.attachments) {
        mail.attachments.forEach(function (att) {
            console.log(att);
            // save att as binary in email.attachments buncket
            var res = mail.to + ".attachments/" + att.contentId;
            // @todo save in Riak (with Link to msg)
            delete att.stream;
            //console.log(att);
            att.content = JSON.stringify(att.content);
        });
    }


    var bucket = mail.to + '.inbox';
    var key =crypto.createHash('sha256').update(mail.msgId).digest('hex');
    // On desactive le search sinon tous les champs vont etre indexés
    riak.saveBucket(bucket, {search: false}, function () {
        // Save mail
        riak.save(bucket, key, mail, function (err, t, l) {
            // Index mail
            riak.search.add(bucket, {id: key, text: mail.text, subject: mail.subject},function(r,t,l){
                console.log("Mail stored and indexed ");
            });
        });
    });
});

/* Writre mail from stdin */
process.stdin.resume();
process.stdin.on('data', function(chunk) {
    mailSize+=chunk.length;
});
process.stdin.pipe(mailparser);


//fs.createReadStream("/home/tmail/dev/samples/mail_text.eml").pipe(mailparser);
 
