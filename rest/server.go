package rest

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Server struct{}

func New() Server {
	return Server{}
}

func (s Server) Run() {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)

	router.Use(middleware.Recoverer)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/processing", s.processing)
	})

	log.Fatal(http.ListenAndServe(":9090", router))
}

func (s Server) processing(w http.ResponseWriter, r *http.Request) {
	var request Job

	defer r.Body.Close()
	err := render.DecodeJSON(r.Body, &request)
	if err != nil {
		render.Render(w, r, InvalidRequest)
		return
	}

	go s.downloadAudio(request)

	render.Status(r, http.StatusOK)
}

func (s Server) downloadAudio(j Job) {
	resp, err := http.Get(j.Link)
	if err != nil {
		log.Printf("[%s] video download error %s", j.JobID, err.Error())
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] video read error %s", j.JobID, err.Error())
		return
	}

	err = ioutil.WriteFile("./"+j.JobID+".avi", data, 0644)
	if err != nil {
		log.Printf("[%s] creation tmp file error %s", j.JobID, err.Error())
		return
	}
	log.Printf("[%s] completed successfully", j.JobID)
}
