package server

import (
	"flag"
	"fmt"
	"net/http"

	"encoding/json"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	log "github.com/golang/glog"
)

type URLList struct {
	Results []string `json:"results"`
}

type Snapshot struct {
	URI  string
	Path string
}

type SnapshotList struct {
	Snapshots []Snapshot
}

func listUrls(c web.C, w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	taskUrls := []string{}

	rows, err := db.Query("SELECT uri FROM runs")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var uri string
		rows.Scan(&uri)
		taskUrls = append(taskUrls, uri)
	}

	urlResults := &URLList{
		Results: taskUrls,
	}

	uriJson, err := json.Marshal(urlResults)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, string(uriJson))
}

func listSnapshots(c web.C, w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	snapshotList := SnapshotList{}

	rows, err := db.Query("SELECT uri,storage_path FROM runs")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var uri string
		var storage string
		rows.Scan(&uri, &storage)
		snap := Snapshot{
			URI:  uri,
			Path: fmt.Sprintf("https://s3.amazonaws.com/mesos-hackathon-bucket/%s", storage),
		}
		snapshotList.Snapshots = append(snapshotList.Snapshots, snap)
	}

	uriJson, err := json.Marshal(snapshotList)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, string(uriJson))
}

func ServeSchedulerAPI(address string, port int) string {
	goji.Get("/whampire_executor", http.FileServer(http.Dir("./executor")))
	goji.Get("/api/url", listUrls)
	goji.Get("/api/snapshots", listSnapshots)
	flag.Set("bind", fmt.Sprintf(":%d", port))
	go goji.Serve()
	return fmt.Sprintf("http://%s:%d/whampire_executor", address, port)
}
