/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

var MailParser = require("mailparser").MailParser,
    mailparser = new MailParser({debug:false, showAttachmentLinks:true, streamAttachments:false}),
    fs = require("fs");

mailparser.on("attachment", function (attachment) {
    console.log(attachment);
    //var output = fs.createWriteStream('/var/www/protecmail.com/Symfony/web/quarantine/' + messageID + '_' + attachment.generatedFileName);
    //attachment.stream.pipe(output);
});


mailparser.on("end", function(mail){
    //jsonmail=mail.clone();
    //var jsonmail=JSON.stringify(mail);
    //var tmail=JSON.parse(jsonmail);
    //console.log(tmail.attachments);
    //console.log(JSON.stringify(mail));
    mail.attachments.forEach(function(att){
        //console.log(att);
        att.content=JSON.stringify(att.content);
       //console.log(att);
    });


    //var copy = new Buffer(JSON.parse(json));
    //console.log(mail);
    var jsonmail=JSON.stringify(mail);
    console.log(jsonmail)

});

fs.createReadStream("/home/toorop/Projects/webmail.pro/dev/sample/mail_pdf.eml").pipe(mailparser);
