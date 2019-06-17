package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanjyoder/sofp"
)

func main() {
	var err error
	templateDir, ok := os.LookupEnv("TEMPLATE_DIR")
	if !ok {
		log.Fatal("TEMPLATE_DIR must be set")
	}

	streamsDir, ok := os.LookupEnv("STREAMS_DIR")
	if !ok {
		log.Fatal("STREAMS_DIR must be set")
	}

	listenPort, ok := os.LookupEnv("LISTEN_PORT")
	if !ok {
		log.Fatal("LISTEN_PORT must be set")
	}

	handler, err := NewStackfarHandler(streamsDir, templateDir)
	if err != nil {
		log.Fatal("failed to create handler:", err)
	}
	http.HandleFunc("/", handler.viewHandler)
	log.Fatal(http.ListenAndServe(":"+listenPort, nil))
}

type StackfarHandler struct {
	streamsDir   string
	templateDir  string
	pageTemplate *template.Template
}

func NewStackfarHandler(streamsDir string, templateDir string) (*StackfarHandler, error) {
	pageTemplateFilepath := filepath.Join(templateDir, "stackoverflow.html")
	pageTemplate, err := template.ParseFiles(pageTemplateFilepath)
	if err != nil {
		return nil, err
	}

	return &StackfarHandler{
		streamsDir:   streamsDir,
		templateDir:  templateDir,
		pageTemplate: pageTemplate,
	}, nil
}

func (h *StackfarHandler) viewHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *StackfarHandler) assetHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *StackfarHandler) loadPage(domain string, id string) (*sofp.Question, error) {
	if len(id) < 1 {
		return nil, fmt.Errorf("id must not be empty")
	}
	subdir := ("000" + id)[len(id):]
	filepath := filepath.Join(h.streamsDir, domain, subdir, id)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	question := &sofp.Question{
		Body: template.HTML("under construction"),
	}
	for scanner.Scan() {
		rowStr := scanner.Text()
		row := sofp.Row{}
		err = json.Unmarshal([]byte(rowStr), &row)
		if err != nil {
			return nil, err
		}
		err = question.AppendRow(&row)
		if err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return question, nil
}
