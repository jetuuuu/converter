package rest

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	timings = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "converter_method_timing",
			Help: "per method time",
		},
		[]string{"method"},
	)
)

type Server struct{}

func New() Server {
	return Server{}
}

func (s Server) Run() {
	prometheus.MustRegister(timings)
	router := chi.NewRouter()

	router.Use(middleware.Recoverer)

	router.Route("/api/v1", func(r chi.Router) {
		router.Use(middleware.RequestID)
		router.Use(middleware.RealIP)
		router.Use(middleware.Logger)

		r.Post("/processing", s.processing)
	})

	router.Handle("/metrics", promhttp.Handler())

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
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[%s] video download error", j.JobID)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] video read error %s", j.JobID, err.Error())
		return
	}

	src := "./" + j.JobID + ".avi"
	defer func() {
		err := os.Remove(src)
		log.Printf("[%s] [INFO] remove tmp video file; Error %s", j.JobID, err.Error())
	}()

	err = ioutil.WriteFile(src, data, 0644)
	if err != nil {
		log.Printf("[%s] creation tmp file error %s", j.JobID, err.Error())
		return
	}

	cmd := exec.Command("ffmpeg", []string{"-i", src, "-q:a", "0", "-map", "a", strings.Replace(src, ".avi", ".mp3", -1)}...)
	if err := cmd.Run(); err != nil {
		log.Printf("[%s] extract audio error %s ", j.JobID, err.Error())
		return
	}

	log.Printf("[%s] completed successfully", j.JobID)
}

func timeTrackMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		timings.WithLabelValues(r.URL.Path).Observe(float64(time.Since(start).Seconds()))
	}
	return http.HandlerFunc(fn)
}
