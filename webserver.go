package stackfar

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/ryanjyoder/sofp"
)

type WebServer struct {
	streamStore    *StreamStore
	templateDir    string
	pageTemplate   *template.Template
	searchTemplate *template.Template
	bindAddress    string
}

func NewWebServer(streamsStore *StreamStore, templateDir string, bindAddress string) (*WebServer, error) {
	pageTemplateFilepath := filepath.Join(templateDir, "question.tpl")
	pageTemplate, err := template.ParseFiles(pageTemplateFilepath)
	if err != nil {
		return nil, err
	}
	searchTemplateFilepath := filepath.Join(templateDir, "search.tpl")
	searchTemplate, err := template.ParseFiles(searchTemplateFilepath)
	if err != nil {
		return nil, err
	}

	return &WebServer{
		streamStore:    streamsStore,
		templateDir:    templateDir,
		pageTemplate:   pageTemplate,
		searchTemplate: searchTemplate,
		bindAddress:    bindAddress,
	}, nil
}

func (h *WebServer) ListenAndServer() error {

	http.HandleFunc("/", h.viewHandler)
	return http.ListenAndServe(h.bindAddress, nil)
}

func (h *WebServer) viewHandler(w http.ResponseWriter, r *http.Request) {
	tokens := strings.Split(strings.Split(r.URL.Path, "?")[0], "/")
	if len(tokens) < 2 {
		log.Println("url path incorrectly formatted:", r.URL.Path)
		http.Error(w, "path incorrectly formated", 400)
		return
	}

	// assume the domain is redirected: https://askubuntu.com/questions/123
	domain := strings.Split(r.Host, ":")[0]
	handler := tokens[1]
	questionID := ""
	if len(tokens) >= 3 {
		questionID = tokens[2]
	}

	// if the domain is not redirect to this host the domain will be in the path, eg:  http://localhost/askubuntu.com/questions/123
	if handler != "questions" && handler != "assets" && handler != "search" {
		if len(tokens) < 3 {
			log.Println("url path incorrectly formatted:", r.URL.Path)
			http.Error(w, "path incorrectly formated", 400)
			return
		}
		domain = tokens[1]
		handler = tokens[2]
		if len(tokens) >= 4 {
			questionID = tokens[3]
		}
	}
	switch handler {
	case "assets":
		// oops actually an asset no a page
		h.assetHandler(w, r)
		return

	case "questions":
		h.questionHandler(w, domain, questionID)

	case "search":
		h.searchHandler(w, r)

	default:
		http.Error(w, "path incorrect format", 400)
	}
}

type Hit struct {
	Title   string
	Href    string
	Summary string
}
type SearchResults struct {
	Results []Hit
	Query   string
}

func (h *WebServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query can't be empty", 400)
	}
	fmt.Println("searching for:", query)
	results, err := h.streamStore.Search(query)
	if err != nil {
		http.Error(w, "internal server error:"+err.Error(), 500)
	}

	hits := make([]Hit, len(results.Hits))
	for i, hit := range results.Hits {
		hits[i].Title = fmt.Sprint(hit.Fields["Title"])
		domain := strings.Split(hit.ID, "/")[0]
		id := strings.Split(hit.ID, "/")[1]
		hits[i].Href = fmt.Sprintf("/%s/questions/%s/", domain, id)
		body := strip.StripTags(fmt.Sprint(hit.Fields["Body"]))
		summarLen := 140
		if len(body) <= 140 {
			summarLen = len(body)
		}
		hits[i].Summary = body[:summarLen]
	}

	h.searchTemplate.Execute(w, SearchResults{
		Results: hits,
		Query:   r.URL.Query().Get("q"),
	})
}

func (h *WebServer) questionHandler(w http.ResponseWriter, domain string, questionID string) {
	//fmt.Println("domain:", domain, "id:", questionID)

	p, err := h.loadPage(domain + "/" + questionID)
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

func (h *WebServer) loadPage(id string) (*sofp.Question, error) {
	deltas, err := h.streamStore.GetStreamDeltas(id)
	if err != nil {
		return nil, err
	}

	return DeltasToQuestion(deltas)
}

func DeltasToQuestion(deltas []*sofp.Row) (*sofp.Question, error) {
	question := &sofp.Question{
		Body: template.HTML("under construction"),
	}

	for i := range deltas {
		err := question.AppendRow(deltas[i])
		if err != nil {
			return nil, err
		}
	}
	return question, nil
}
