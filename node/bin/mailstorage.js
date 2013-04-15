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



var uuid = require('node-uuid'),
    sanitize = require('validator').sanitize,
    crypto = require('crypto'),
    jquery = require('jquery');


var config = require('../config.json');

// Riak
var riak = require("riak-js").getClient({host: config.riak.apiHost, port: config.riak.apiPort, debug: true });

// Expose to the world
module.exports.MailStorage = MailStorage;

function MailStorage(mail) {
    mail = mail || null;
}

MailStorage.prototype.dump = function (callback) {
    //console.log(mail);
    if (typeof callback == "function") {
        callback('toto est dans la place');
    }
};


MailStorage.prototype.store = function (mail, callback) {
    //var mail = mail;

    // envelope sender
    //mail.from = "toorop@toorop.fr";
    mail.from = mail.from[0].address;

    // envelope From
    /*mail.to = "toorop@tmail.io";
     if (!mail.to) {
     mail.to = mail.headers.to;
     }*/
    //mail.to=mail.to[0].address;
    mail.to = 'toorop@toorop.fr';

    // subject
    mail.subject = mail.headers['subject'];
    if (!mail.subject) {
        mail.subject = "no_subject";
    }


    // date
    if (mail.headers['dated']) {
        mail.date = Date.parse(mail.headers['date']);
    } else mail.date = new Date().getTime();
    mail.date = parseInt(mail.date);


    // msgId
    mail.msgId = mail.headers['message-id'];
    if (mail.msgId) { // remove <>
        mail.msgId = mail.msgId.substr(1, mail.msgId.length - 2);
    } else {
        mail.msgId = uuid.v1();
    }

    // @todo : il faut tester les filtres

    // labels
    mail.labels = [];
    // inbox
    mail.labels.push('inbox');
    mail.labels.push('mailing/tt');

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


    // Text a indexer
    var indexText = '';
    mail.text = sanitize(mail.text).trim();
    if (mail.text.length > 0) {
        indexText = mail.text;
    } else {
        console.log("mail.text est vide :" + mail.text);
        if (mail.html && mail.html.length > 0) {
            indexText = jquery(mail.html).text();
        }
    }

    // on doit nettoyer les attachements
    /*if (mail.attachments) {
     mail.attachments.forEach(function (att) {
     console.log(att);
     // save att as binary in email.attachments buncket
     var res = mail.to + ".attachments/" + att.contentId;
     // @todo save in Riak (with Link to msg)
     delete att.stream;
     //console.log(att);
     att.content = JSON.stringify(att.content);
     });
     }*/

    // Save and index mail
    var bucket = mail.to + '.inbox';
    var key = crypto.createHash('sha1').update(mail.msgId).digest('hex');
    //var mail = mail;


    riak.saveBucket(bucket, {search: false}, function (err) {
            if (err) throw err;
            riak.save(bucket, key, mail, function (err, t, l) {
                if (err) throw err;
                //console.log("Riak save", err, t, l);
                // @toto add indexes : labels
                var labels = "  ";
                for (var i in mail.labels) {
                    labels = labels + mail.labels[i] + "  ";
                }

                // index
                var index = {};
                index.id = key;
                //index.text = indexText;
                indexText = sanitize(indexText).trim();
                if (indexText.length > 0) {
                    index.text = indexText;
                }
                mail.subject = sanitize(mail.subject).trim();
                if (mail.subject.length > 0) {
                    index.subject = mail.subject;
                }
                index.indexed_num = 1;
                index.date_date = mail.date;
                index.labels = labels;
                index.seen_num = 0;
                index.att_num = 0;

                // clean index.text
                /*var tmp = index.text.indexOf('\u0003');
                 console.log(tmp);*/
                if (index.text) {
                    // tolower
                    index.text = index.text.toLowerCase();
                    // remove useless char
                    index.text = index.text.replace(/[\u0000-\u001F]/g, ' ');
                    index.text = index.text.replace(/[\u0021-\u002F]/g, ' ');
                    index.text = index.text.replace(/[\u003A-\u003F]/g, ' ');
                    // accent éèê -> e
                    index.text = index.text.replace(/[\u00E8-\u00EB]/g, 'e');
                    // ô -> o
                    index.text = index.text.replace(/[\u00F2-\u00F6]/g, 'o');
                    var tokens = index.text.split(' ');
                    var keep = [];
                    for (var i in tokens) {
                        if (tokens[i].length > 2) {
                            keep.push(tokens[i]);
                        }
                    }
                    delete(tokens);

                    index.text = keep.join(" ");
                    if (!index.text) delete index.text;
                    if (index.text && index.text.length < 5) {
                        delete index.text;
                    }
                    delete keep;
                }

                //riak.search.add(bucket, {id: key, text: mail.text, subject: mail.subject, indexed_num: 1, date_date: mail.date, labels: labels, seen_num: 0}, function (err, t, l) {
                riak.search.add(bucket, index, function (err, t, l) {
                    if (err) {
                        console.log("----------------------------------------------------");
                        console.log("INDEXATION FAILURE");
                        console.log("Key : " + index.id);
                        console.log("Text : " + index.text);
                        console.log("Subject : " + index.subject);
                        console.log("Date : " + index.date_date);
                        console.log(err);
                        console.log("----------------------------------------------------");
                        process.exit(0);

                    }
                    console.log("Mail : " + key + " indexed\n");
                    //console.log(typeof callback);
                    if (typeof callback == "function") {
                        callback('toto est dans la place');
                    }
                });
            });
        }
    )
    ;
}
;

/**
 ----------------------------------------------------
 INDEXATION FAILURE
 Key : d81f63c574a5d1d69a3ff6e82ba90b7e547139ca
 Text :
 Subject : [tt] Spéciale Toorop...
 Date : 1365756118870
 { [Error: Unable to parse request: {expected_binaries,<<"text">>,[]}]
     message: 'Unable to parse request: {expected_binaries,<<"text">>,[]}',
         statusCode: 400 }
 ----------------------------------------------------

 */