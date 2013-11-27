#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */


var bucket=process.argv[2];
if(!bucket){
    console.log("Usage : ./deleteBucket.js bucket");
    process.exit(0);
}


var config=require('../config.json');
var riak = require("riak-js").getClient({host: config.riak.apiHost, port: config.riak.apiPort, debug: true });
var querystring = require("querystring");

riak.keys(bucket, { keys: 'stream' }).on('keys', function(msgid){
    //console.log(typeof msgid,msgid);
    msgid.forEach(function(id){
        console.log(id);
        /*riak.get(bucket, id, function (err, mail, meta) {
            if (err) throw err;
            console.log(mail);
            //console.log(meta);
        })*/
    });

}).start();