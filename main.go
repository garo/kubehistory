
package main

import (
//	"github.com/golang/glog"
  "flag"
	"os"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/tools/cache"

)


func main() {
	flag.Set("logtostderr", "true")

	clientset, err := getClient()
	if err != nil {
		panic(err)
	}

	snapshotter := NewSnapshotter(clientset)

	var webserver *Webserver

	psql_uri := os.Getenv("PSQL_URI")
	if psql_uri != "" {
		sqlStorage, err := NewSqlStorage(psql_uri)
		err = sqlStorage.Init()
		if err != nil {
			panic(err)
		}
		snapshotter.SetPersistentStorage(sqlStorage)
		webserver = NewWebserver(sqlStorage)
	}

	go webserver.Start()
	snapshotter.Init()



  select {}

}