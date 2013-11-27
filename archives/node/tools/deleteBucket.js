#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */


var bucket = process.argv[2];
if (!bucket) {
    console.log("Usage : ./deleteBucket.js bucket");
    process.exit(0);
}


var config = require('../config.json');
var riak = require("riak-js").getClient({host: config.riak.apiHost, port: config.riak.apiPort, debug: true });
var querystring = require("querystring");

riak.saveBucket(bucket, {search: true}, function () {
    riak.keys(bucket, { keys: 'stream' }).on('keys',function (keys) {
        //console.log(keys);
        keys.forEach(function (key) {
            // on commence par supprimer l'index
            console.log("Remove index for key : "+key);
            riak.search.remove(bucket, {id: key}, function (err) {
                if (err) throw err;
                console.log("Index for key "+key+" removed.");
                // puis les data
                riak.remove(bucket, key, function (err) {
                    if (err) console.log(err);
                    else console.log(key + ' fully removed');
                })
            })
        });
    }).start();
});