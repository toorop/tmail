#!/usr/bin/env node

/**
 * (c) Stéphane Depierrepont <toorop@toorop.fr>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

// curl http://10.10.0.1:8098/solr/toorop@tmail.io.inbox/select?q=Christophe

var config = require('../config.json');

var crypto = require('crypto');

var key =crypto.createHash('sha256').update('OTcxNTEwMQAC1128071Y23BAMTM2NDg1Mjk2OTg1OTc0@www.integration-paiement-en-ligne.com').digest('hex');


var bucket = 'toorop@tmail.io.inbox';

var data = {
    text: "J'ai vérifié l'adresse existe de notre côté et il y a un all dans les mails valides côté protecmail.\nje pense que les mails de relance bouncés sont dus à une erreur de notre part effectivement.",
    subject: "[PROTECMAIL][SUPPORT] problème de fonctionnement de la whiteliste",
    bidule: "de doit pas etre indexe cassandre detre"
};


var riak = require("riak-js").getClient({host: config.riak.apiHost, port: config.riak.apiPort, debug: true });

/*
riak.saveBucket(bucket, {search: false}, function () {
    console.log("OPTION CHANGED");
    riak.save(bucket, key, data, function (err, t, l) {
        //console.log(err, t, l);
        console.log("Riak save", err, t, l);
        riak.search.add(bucket, {id: key, text: data.text, subject: data.subject, prop:"ce sont les prop de"},function(r,t,l){
            console.log("INDEXED",r,t,l);
        });
    });
});
*/



//riak.search.add(bucket, {id: 'aezerty', text: 'la caverne des trois', prop:"ce sont les prop de"});


riak.search.find(bucket, 'text:Christophe', function (err, t, l) {
 console.log(err);
 console.log(t);
 console.log(t.docs[0].id+"\n");
 });

var map = function(v, keydata, args) {
    //v is the full value of data kept in riak
    //your data plus meta data
    //check riak wiki m/r page for an example of what 'v' looks like
    if (v.values) {
        //init an empty return array
        var ret = [];
        //set 'o' (for object) equal to the data portion
        //Riak.mapValuesJson is an internal riak js func
        o = Riak.mapValuesJson(v)[0];
        //delete o.headers;
        //interesting part for the sorting.
        //pull the last modified datestamp string out of the meta data
        //and turn it into an int
        o.lastModifiedParsed = Date.parse(v["values"][0]["metadata"]["X-Riak-Last-Modified"]);
        //i also return the key just for good measure which i use elsewhere in my app
        o.key = v["key"];
        //push the 'o' object into the ret array
        ret.push(o);
        return ret;
    } else { //if no value return an empty array
        return [];
    }
};

var reduceDescending = function ( v , args ) {
    //by default sort() sorts elements ascending, alpha.
    //we want numeric sort so we provide a numeric sort function
    //there is a riak builtin func but it expects an array of numeric values,
    //not the numeric nested in an object
    //here I return in DESC order, if you want ASC order rewrite return to 'a-b'
    v.sort ( function(a,b) { return b['lastModifiedParsed'] - a['lastModifiedParsed'] } );
    return v;
};

/*riak.mapreduce.search(bucket, 'text:participation AND subject:need').map(map).reduce(['Riak.filterNotFound',reduceDescending]).run(function(r,t,l){
   console.log(t);
});*/




//riak.mapreduce.search(bucket, 'mail').map('Riak.mapValues').run();
