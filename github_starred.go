package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"time"

	"golang.org/x/net/proxy"
)

type Repo struct {
	Name        string `json:"name"`
	HtmlUrl     string `json:"html_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
}

type RepoSlice []*Repo

func (r RepoSlice) Len() int           { return len(r) }
func (r RepoSlice) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r RepoSlice) Less(i, j int) bool { return r[i].Name < r[j].Name }

var readme = make(map[string]*RepoSlice)
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

	var projectNames []string
	for k, v := range readme {
		projectNames = append(projectNames, k)
		sort.Sort(v)
	}
	sort.Strings(projectNames)

	for _, n := range projectNames {
		fmt.Printf("# %s (%d)\n\n", n, len(*readme[n]))
		for _, val := range *readme[n] {
			fmt.Printf("* [%s](%s) - %s\n", val.Name, val.HtmlUrl, val.Description)
		}
		fmt.Printf("\n\n")
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
		repoList, ok := readme[repo.Language]
		if !ok {
			repoList = &RepoSlice{}
			readme[repo.Language] = repoList
		}
		*repoList = append(*repoList, repo)
	}

	if len(nextLink) != 0 && nextLink[1] != "" {
		fetch(nextLink[1])
	}
}
