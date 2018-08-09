package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type Repo struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HtmlUrl     string `json:"html_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Owner       string
}

type RepoSlice struct {
	repos  []*Repo
	client *http.Client
}

func (r RepoSlice) Len() int      { return len(r.repos) }
func (r RepoSlice) Swap(i, j int) { r.repos[i], r.repos[j] = r.repos[j], r.repos[i] }
func (r RepoSlice) Less(i, j int) bool {
	if r.repos[i].Language == r.repos[j].Language {
		if r.repos[i].Owner == r.repos[j].Owner {
			return r.repos[i].Name < r.repos[j].Name
		}
		return r.repos[i].Owner < r.repos[j].Owner
	}
	return r.repos[i].Language < r.repos[j].Language
}

func (r *RepoSlice) print() {
	sort.Sort(r)
	var language string
	for _, repo := range r.repos {
		if language != repo.Language {
			fmt.Printf("\n\n# %s\n\n", repo.Language)
			language = repo.Language
		}
		fmt.Printf("* [%s](%s) - %s (%s)\n", repo.Name, repo.HtmlUrl, repo.Description, repo.Owner)
	}
}

var next = regexp.MustCompile(`<(https://api\.github\.com/user/\d+/starred\?per_page=\d+&page=\d+)>; rel="next"`)

func main() {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, proxy.Direct)
	if err != nil {
		panic(err)
	}

	var repositories = &RepoSlice{
		client: &http.Client{
			Transport: &http.Transport{
				TLSHandshakeTimeout: 30 * time.Second,
				Dial:                dialer.Dial,
			},
		},
	}
	repositories.fetch("https://api.github.com/users/litgh/starred?per_page=100")
	repositories.print()
}

func (r *RepoSlice) fetch(url string) {
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := r.client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, _ := ioutil.ReadAll(resp.Body)
	var repos []*Repo
	json.Unmarshal(b, &repos)

	for _, repo := range repos {
		repo.Owner = strings.Split(repo.FullName, "/")[0]
		r.repos = append(r.repos, repo)
	}

	link := next.FindStringSubmatch(resp.Header.Get("Link"))
	if len(link) != 0 && link[1] != "" {
		r.fetch(link[1])
	}
}
