/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */


var Imap = require('imap'),
    inspect = require('util').inspect;

var imap = new Imap({
    user: 'toorop@toorop.fr',
    password: 'murphymurphy',
    host: 'toorop.dedimail.eu',
    port: 143,
    secure: false
});

function show(obj) {
    return inspect(obj, false, Infinity);
}

function die(err) {
    console.log('Uh oh: ' + err);
    process.exit(1);
}

function openInbox(cb) {
    imap.connect(function (err) {
        if (err) die(err);
        imap.openBox('INBOX.Mailing Lists.TT', true, cb);
    });
}

function getBoxes() {
    imap.connect(function (err) {
        if (err) die(err);
        imap.getBoxes('INBOX', function (err, boxes) {
            if (err) die(err);
            console.log(show(boxes));

        });
    });
}
//getBoxes();


openInbox(function (err, mailbox) {
    if (err) die(err);
    imap.on("mail", function (message) {
        console.log('New mail: ', message);
    });

    imap.search([ 'ALL', ['SINCE', 'Mars 25, 2013'] ], function (err, results) {
        if (err) die(err);
        imap.fetch(results,
            {
                headers: true,
                body: true,
                cb: function(fetch) {
                    var mail={};
                    fetch.on('message', function(msg) {
                        console.log('Saw message no. ' + msg.seqno);
                        var body = '';
                        msg.on('headers', function(hdrs) {
                            //console.log('Headers for no. ' + msg.seqno + ': ' + show(hdrs));
                            mail.header=hdrs;
                        });
                        msg.on('data', function(chunk) {
                            body += chunk.toString('utf8');
                        });
                        msg.on('end', function() {
                            mail.body=body;
                            console.log('Finished message no. ' + msg.seqno);
                            console.log('UID: ' + msg.uid);
                            console.log('Flags: ' + msg.flags);
                            console.log('Date: ' + msg.date);
                            console.log(JSON.stringify(mail));
                        });
                    });
                }
            }, function (err) {
                if (err) throw err;
                console.log('Done fetching all messages!');
                imap.logout();
            }
        );
    });
});
