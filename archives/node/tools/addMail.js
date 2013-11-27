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

var mailSize=0; // Size of the mail in Octet

// Mail storage
var MailStorage=require('../bin/mailstorage').MailStorage;

// Mailparser
var MailParser = require("mailparser").MailParser,
    mailparser = new MailParser({debug: false, showAttachmentLinks: true, streamAttachments: true});


/**
 * Parse & save & index mail
 */
mailparser.on("end", function (mail) {
    mail.size=mailSize;
    var mailstorage= new MailStorage();
    mailstorage.store(mail);
});

// Read mail from stdin
process.stdin.resume();

process.stdin.on('data', function(chunk) {
    mailSize+=chunk.length;
});

process.stdin.pipe(mailparser);

 
