#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

var uuid = require('node-uuid');


var MailParser = require("mailparser").MailParser,
    mailparser = new MailParser({debug: false, showAttachmentLinks: true, streamAttachments: true}),
    fs = require("fs");

mailparser.on("attachment", function (attachment) {
    //console.log(attachment);
    //var output = fs.createWriteStream('/var/www/protecmail.com/Symfony/web/quarantine/' + messageID + '_' + attachment.generatedFileName);
    //attachment.stream.pipe(output);
});


mailparser.on("end", function (mail) {
    //var msgId=mail.message-id
    var msgId = mail.headers['message-id'];
    if (msgId) { // remove <>
        msgId = msgId.substr(1, msgId.length - 2);
    } else {
        msgId=uuid.v1();
    }


    console.log(msgId);
    process.exit(0);

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
        } else mail.threadIndex = "";
    }

    // prev & next
    if (mail.inReplyTo) {
        mail.prev = mail.inReplyTo;
        delete mail.inReplyTo;
    } else mail.prev = "";
    mail.next = "";

    // on doit netoyer les attachements
    if (mail.attachments) {
        mail.attachments.forEach(function (att) {
            console.log(att);
            // save att as binary in email.attachments buncket
            var res = mail.to + ".attachments/" + att.contentId;
            att.msgId


            delete att.stream;
            console.log(att);
            att.content = JSON.stringify(att.content);
        });
    }


    //var copy = new Buffer(JSON.parse(json));
    //console.log(mail);
    var jsonmail = JSON.stringify(mail);
    //console.log(jsonmail)

});
// mail text brut
//fs.createReadStream("/home/toorop/Projects/webmail.pro/dev/samples/mail_text.eml").pipe(mailparser);
// mail + pj pdf
process.stdin.resume();
process.stdin.pipe(mailparser);


//fs.createReadStream("/home/tmail/dev/samples/mail_pdf.eml").pipe(mailparser);
