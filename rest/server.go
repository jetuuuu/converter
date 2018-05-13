package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/jetuuuu/converter/config"

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

type Server struct {
	cfg      config.ConfigReader
	nodeAddr string
	token    string
}

func New(cfgReader config.ConfigReader) Server {
	return Server{
		cfg: cfgReader,
	}
}

func (s Server) Run() error {
	if err := s.register(); err != nil {
		return err
	}
	router := chi.NewRouter()

	router.Use(middleware.Recoverer)

	router.Route("/api/v1", func(r chi.Router) {
		router.Use(middleware.Recoverer)
		router.Use(middleware.RequestID)
		router.Use(middleware.RealIP)
		router.Use(middleware.Logger)

		r.Post("/processing", s.processing)
	})

	router.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(":8080", router)
	log.Fatal(err)
	return err
}

func (s *Server) register() error {
	prometheus.MustRegister(timings)
	cfg := s.cfg.Read()
	var nodes []string
	for {
		n := cfg.Apis.Next()
		resp, err := http.Get("http://" + n.Adress + "/api/v1/converter/register")
		if err != nil {
			log.Printf("[WARN] got error while register in node %s", n.Adress)
			for _, a := range nodes {
				if a == n.Adress {
					return fmt.Errorf("all nodes down")
				}
			}
			continue
		}

		request := make(map[string]string)
		render.DecodeJSON(resp.Body, &request)
		resp.Body.Close()

		s.token = request["token"]
		s.nodeAddr = "http://" + n.Adress + "/api/v1/converter"
		return nil
	}
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
	var (
		err       error
		audioPath string
	)
	defer func() {
		status := "done"
		if err != nil {
			status = "fail"
		}
		data, _ := json.Marshal(map[string]string{
			"job_id": j.JobID,
			"status": status,
		})

		req, _ := http.NewRequest(http.MethodPost, s.nodeAddr+"/job", bytes.NewReader(data))
		req.Header.Set("Authorization", "BEARER "+s.token)
		client := http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("[WARN] api node does not response; remove file %s", audioPath)
			os.Remove(audioPath)
		}
	}()

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

	src := "/audio/" + j.JobID + ".avi"
	defer func() {
		err := os.Remove(src)
		if err != nil {
			log.Printf("[%s] [INFO] remove tmp video file; Error %s", j.JobID, err.Error())
		}
	}()

	err = ioutil.WriteFile(src, data, 0644)
	if err != nil {
		log.Printf("[%s] creation tmp file error %s", j.JobID, err.Error())
		return
	}

	audioPath = strings.Replace(src, ".avi", ".mp3", -1)
	cmd := exec.Command("ffmpeg", []string{"-i", src, "-q:a", "0", "-map", "a", audioPath}...)
	err = cmd.Run()
	if err != nil {
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
