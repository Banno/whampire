package server

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func listUrls(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
}

func ServeSchedulerAPI(address string, port int) string {
	goji.Get("/scraper_executor", http.FileServer(http.Dir(".")))
	goji.Get("/api/url", listUrls)
	flag.Set("bind", fmt.Sprintf(":%d", port))
	go goji.Serve()
	return fmt.Sprintf("http://%s:%d/scraper_executor", address, port)
}
