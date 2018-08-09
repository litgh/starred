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

type RepoSlice []*Repo

func (r RepoSlice) Len() int      { return len(r) }
func (r RepoSlice) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r RepoSlice) Less(i, j int) bool {
	if r[i].Language == r[j].Language {
		if r[i].Owner == r[j].Owner {
			return r[i].Name < r[j].Name
		}
		return r[i].Owner < r[j].Owner
	}
	return r[i].Language < r[j].Language
}

var repositories RepoSlice
var next = regexp.MustCompile(`<(https://api\.github\.com/user/\d+/starred\?per_page=\d+&page=\d+)>; rel="next"`)
var client *http.Client

func main() {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, proxy.Direct)
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		TLSHandshakeTimeout: 30 * time.Second,
		Dial:                dialer.Dial,
	}
	client = &http.Client{
		Transport: transport,
	}

	fetch("https://api.github.com/users/litgh/starred?per_page=100")

	sort.Sort(repositories)

	var language = struct {
		Name string
	}{}
	for _, repo := range repositories {
		if language.Name != repo.Language {
			fmt.Printf("\n\n# %s\n\n", repo.Language)
			language.Name = repo.Language
		}
		fmt.Printf("* [%s](%s)\n\n", repo.FullName, repo.HtmlUrl)
		fmt.Println(">", repo.Description)
	}
}

func fetch(url string) {
	req, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	links := resp.Header.Get("Link")
	nextLink := next.FindStringSubmatch(links)

	b, _ := ioutil.ReadAll(resp.Body)
	var repos []*Repo
	json.Unmarshal(b, &repos)

	for _, repo := range repos {
		repo.Owner = strings.Split(repo.FullName, "/")[0]
		repositories = append(repositories, repo)
	}

	if len(nextLink) != 0 && nextLink[1] != "" {
		fetch(nextLink[1])
	}
}
