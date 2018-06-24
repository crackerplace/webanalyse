// Package main spwans a server daemon and listens for analyse requests.
package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/Sirupsen/logrus"
	"github.com/stefantalpalaru/pool"
)

var errResponseNot200 = errors.New("Response is not 200")

const errMsgEmptyURL string = "Hey, are you sure you want to analyse a website which doesn't have a url ? Please enter one"
const errMsgGeneric = "We have some issue in accessing the website.Please try again later"

// LinkInfo encapsulates the count of all the different links in the page.
type LinkInfo struct {
	internal    int
	external    int
	inaccesible int
}

// AnalyseResponse holds the analysis summary of the url.
type AnalyseResponse struct {
	URL               string
	Version           string
	Title             string
	Headings          map[string]int
	InternalLinks     int
	ExternalLinks     int
	InaccessibleLinks int
	HasLogin          string
}

// analyser holds a document which needs to be analyzed.
type analyser struct {
	url string
	doc *goquery.Document
}

// ErrorResponse holds the error message on any error.
type ErrorResponse struct {
	Message string
}

func analyseHandler() func(w http.ResponseWriter, r *http.Request) {
	errorTmpl := template.Must(template.ParseFiles("public/error.html"))
	analyseTmpl := template.Must(template.ParseFiles("public/analyse.html"))
	// Use a closure to embed templates as we want to initialize them once.
	// Ofcourse we can use init function, but global state is something that I don't prefer.
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("url") == "" || strings.TrimSpace(r.Form.Get("url")) == "" {
			errorTmpl.Execute(w, ErrorResponse{Message: errMsgEmptyURL})
			return
		}
		url := r.Form.Get("url")
		doc, err := getDocument(url)
		if err != nil {
			var msg string
			switch err.(type) {
			case *notOkResponse:
				msg = fmt.Sprintf("Website did not respond properly, errorcode is: %d", err.(*notOkResponse).code)
			default:
				msg = fmt.Sprintf("Could not access the website: %s, error is: %s ", url, err)
			}

			log.Error("error while getting document for the url: ", url, ", error is: ", err)
			errorTmpl.Execute(w, ErrorResponse{Message: msg})
			return
		}
		an := analyser{url: url, doc: doc}
		links := an.linkInfo()
		hasLogin := "No"
		if an.hasLogin() {
			hasLogin = "Yes"
		}
		analyzeResp := AnalyseResponse{
			URL:               url,
			Version:           an.version(),
			Title:             an.title(),
			Headings:          an.headings(),
			InternalLinks:     links.internal,
			ExternalLinks:     links.external,
			InaccessibleLinks: links.inaccesible,
			HasLogin:          hasLogin,
		}
		analyseTmpl.Execute(w, analyzeResp)
		log.Info("Summary generated for url: ", url)
	}
}

func (an *analyser) title() string {
	//handle empty title or version
	return an.doc.Find("title").Text()
}

func (an *analyser) version() string {
	// get the doctype as the html version
	// https://www.w3.org/QA/2002/04/valid-dtd-list.html
	return goquery.NodeName(an.doc.Find(":root"))
}

func (an *analyser) headings() map[string]int {
	var headings = map[string]int{}
	an.doc.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		tag := goquery.NodeName(s)
		headings[tag]++
	})
	return headings
}

func makeRequest(method string, url string, timeoutDuration time.Duration) (*http.Response, error) {
	timeout := time.Duration(timeoutDuration)
	client := http.Client{
		Timeout: timeout,
	}
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Warn("not able to make a request to the url: ", url)
		return nil, err
	}
	request.Header.Set("User-Agent", "Test bot for home24. contact at admin@domain.com")
	return client.Do(request)
}

func accessible(args ...interface{}) interface{} {
	url := args[0].(string)
	resp, err := makeRequest("HEAD", url, 10*time.Second)
	if err != nil {
		log.Warn("inaccessible url: ", url, " with err: ", err)
		return false
	} else if resp != nil {
		status := resp.StatusCode
		if status >= 400 {
			log.Warn("inaccessible url: ", url, " with statuscode: ", status)
			return false
		}
	}
	return true
}

// Few assumptions below
// Ignore if anchor link has invalid url i.e thows an error when parse by url.Parse()
// Ignore duplicates
// Ignore non http urls
// Count as internal link when url is a relative url(url.Parse returns scheme and host as empty) or when url points to same domain
// Count as external link when scheme is http or https and different domain than the url domain.I am considering even subdomain to be different.
func (an *analyser) linkInfo() LinkInfo {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus) // this anyways is the default even if we dnt set explicitly
	mypool := pool.New(cpus)
	mypool.Run()
	var host string
	u, err := url.Parse(an.url)
	if err == nil {
		host = u.Host
	}
	seenUrls := map[string]bool{}
	internalLinks, externalLinks, inaccesibleLinks := 0, 0, 0
	// find all anchor tags
	an.doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if (exists) && !(seenUrls[href]) {
			seenUrls[href] = true
			u, err := url.Parse(href)
			if err == nil {
				if u.Scheme == "" || u.Host == "" {
					if !strings.HasPrefix(href, "mailto:") {
						internalLinks++
					}
				} else if (u.Scheme == "http") || (u.Scheme == "https") { //ignore non http urls
					if host == u.Host {
						internalLinks++
					} else {
						externalLinks++
						// add the link to the pool to check if its accessible
						mypool.Add(accessible, href)
					}
				}
			} else {
				log.Warn("invalid url : ", href)
			}
		}
	})
	for {
		job := mypool.WaitForJob()
		if job == nil {
			break
		}
		if job.Result == nil {
			log.Warn("job result is nil")
			continue
		} else {
			if !job.Result.(bool) {
				inaccesibleLinks++
			}
		}
	}
	mypool.Stop()
	return LinkInfo{internal: internalLinks, external: externalLinks, inaccesible: inaccesibleLinks}
}

func (an *analyser) hasLogin() bool {
	s := an.doc.Find("input[type=password]")
	if len(s.Nodes) > 0 {
		return true
	}
	return false
}

type notOkResponse struct {
	code int
}

func (e *notOkResponse) Error() string {
	return fmt.Sprintf("status code is %d: ", e.code)
}

func getDocument(url string) (*goquery.Document, error) {
	// Create HTTP client with timeout
	resp, err := makeRequest("GET", url, 20*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Create a goquery document from the HTTP response
	if resp.StatusCode != 200 {
		return nil, &notOkResponse{code: resp.StatusCode}
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	return doc, err
}
