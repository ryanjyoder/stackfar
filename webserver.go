package stackfar

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ryanjyoder/sofp"
)

type WebServer struct {
	streamStores map[string]*StreamStore
	templateDir  string
	pageTemplate *template.Template
	bindAddress  string
}

func NewWebServer(streamsStores map[string]*StreamStore, templateDir string, bindAddress string) (*WebServer, error) {
	pageTemplateFilepath := filepath.Join(templateDir, "stackoverflow.html")
	pageTemplate, err := template.ParseFiles(pageTemplateFilepath)
	if err != nil {
		return nil, err
	}

	return &WebServer{
		streamStores: streamsStores,
		templateDir:  templateDir,
		pageTemplate: pageTemplate,
		bindAddress:  bindAddress,
	}, nil
}

func (h *WebServer) ListenAndServer() error {

	http.HandleFunc("/", h.viewHandler)
	return http.ListenAndServe(h.bindAddress, nil)
}

func (h *WebServer) viewHandler(w http.ResponseWriter, r *http.Request) {
	tokens := strings.Split(r.URL.Path, "/")
	if len(tokens) < 3 {
		log.Println("url path incorrectly formatted:", r.URL.Path)
		http.Error(w, "path incorrectly formated", 400)
		return
	}

	// assume the domain is redirected: https://askubuntu.com/questions/123
	domain := strings.Split(r.Host, ":")[0]
	questionID := tokens[2]

	// if the domain is not redirect to this host the domain will be in the path, eg:  http://localhost/askubuntu.com/questions/123
	if tokens[1] != "questions" {
		if len(tokens) < 4 {
			log.Println("url path incorrectly formatted:", r.URL.Path)
			http.Error(w, "path incorrectly formated", 400)
			return
		}
		domain = tokens[1]
		questionID = tokens[3]
	}
	if questionID == "assets" {
		// oops actually an asset no a page
		h.assetHandler(w, r)
		return

	}
	fmt.Println("domain:", domain, "id:", questionID)

	p, err := h.loadPage(domain, questionID)
	if err != nil {
		log.Println("error loading page data:", err)
		http.Error(w, err.Error(), 404)
		return
	}

	h.pageTemplate.Execute(w, p)
}

func (h *WebServer) assetHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/assets/")
	if len(pathParts) < 2 {
		log.Println("url path incorrectly formatted:", r.URL.Path)
		http.Error(w, "path incorrectly formated", 400)
		return
	}
	path := filepath.Clean(strings.Join(pathParts[1:], "/"))
	path = filepath.Join(h.templateDir, "static", path)
	http.ServeFile(w, r, path)
	return
}

func (h *WebServer) loadPage(domain string, id string) (*sofp.Question, error) {
	store, ok := h.streamStores[domain]
	if !ok {
		return nil, fmt.Errorf("domain not found")
	}

	deltas, err := store.GetStreamDeltas(domain, id)
	if err != nil {
		return nil, err
	}
	question := &sofp.Question{
		Body: template.HTML("under construction"),
	}

	for i := range deltas {
		err = question.AppendRow(deltas[i])
		if err != nil {
			return nil, err
		}
	}

	return question, nil
}
