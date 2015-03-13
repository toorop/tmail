package core

import ()

type routemap struct {
	Host  string
	Route string
}

type smtproute struct {
	Name        string
	LocalAddrs  string
	remoteAddrs string
}

/*
// getRouteToHost return a smtproute to relay mail to host host
// routes are stored in routemap collection
func getRouteToHost(host string) (route smtproute, err error) {
	mongo, err := getMgoSession()
	if err != nil {
		return
	}
	var routeMap routemap
	c := mongo.DB(Config.StringDefault("mongo.db", "tmail")).C("routemap")
	err = c.Find(bson.M{"host": host}).One(&routeMap)
	if err != nil {
		if err != mgo.ErrNotFound {
			return
		}
		// on va utiliser les MX
		var mxs []*net.MX
		route.Name = "mx"
		route.LocalAddrs = "default"
		mxs, err = net.LookupMX(host)
		// TODO handle err
		if err != nil {
			TRACE.Fatalln(err)
		}

		// TODO if no MX test A record

		for _, mx := range mxs {
			route.remoteAddrs += fmt.Sprintf("%s&", mx.Host[:len(mx.Host)-1])
		}
		// remove trailing &
		route.remoteAddrs = route.remoteAddrs[:len(route.remoteAddrs)-1]

		TRACE.Println(route, err)

	}
	TRACE.Println(err)

	return
}
*/
