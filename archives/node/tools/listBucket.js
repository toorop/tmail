#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

var config=require('../config.json');

var bucket='toto';

var riak = require("riak-js").getClient({host: config.riak.apiHost, port: config.riak.apiPort, debug: true });


riak.buckets(function(err, buckets, meta) {
    if (err) throw (err);
    console.log("Response code : "+meta.statusCode);
    console.log(buckets);

});