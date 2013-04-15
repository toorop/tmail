#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

EventEmitter = require('events').EventEmitter;
var evtManager = new EventEmitter();

var corpusPath = "/home/tmail/dev/samples/corpus/cur";
var maxSample = 6000;
var maxOpenFile = 100;
var fileOpened = 0;
var queueFiles = [];

var fs = require('fs');


// Mail storage
var MailStorage = require('../bin/mailstorage').MailStorage;

// Mailparser
var MailParser = require("mailparser").MailParser;

var storeMail = function (mailPath) {
    var mailSize = 0;

    var mailparser = new MailParser({debug: false, showAttachmentLinks: true, streamAttachments: true});

    mailparser.on("end", function (mail) {
        mail.size = mailSize;
        var mailstorage = new MailStorage();
        mailstorage.store(mail,function() {
            //console.log(fileOpened, queueFiles.length);
            fileOpened = fileOpened - 1;
            evtManager.emit("processNewMail");
        });
        /*mailstorage.dump(function() {
            fileOpened = fileOpened - 1;
            console.log(fileOpened, queueFiles.length);
            evtManager.emit("processNewMail");
        });*/
    });

    var mailStream = fs.createReadStream(mailPath);
    mailStream.on('data', function (chunk) {
        mailSize += chunk.length;
    });
    mailStream.pipe(mailparser);

};

evtManager.on("processNewMail", function () {
    if (queueFiles.length > 0 && fileOpened < maxOpenFile) {
        fileOpened = fileOpened + 1;
        storeMail(queueFiles.pop());
    }
});

evtManager.on("newMail", function (fpath) {
    queueFiles.push(fpath);
    evtManager.emit("processNewMail");
});


/*
 Main
 */

// list dir
fs.readdir(corpusPath, function (err, files) {
    if (err) throw err;
    files = files.slice(0, maxSample);
    files.forEach(function (file) {
        var fpath = corpusPath + "/" + file;
        //console.log(fpath);
        evtManager.emit('newMail', fpath);
    })


});


