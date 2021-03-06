package Gist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MaximeD/gost/json"
	"github.com/MaximeD/gost/utils"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
 * Show all gists for a user.
 */
func List(url string, accessToken string) {
	if accessToken != "" {
		url = url + "?access_token=" + accessToken
	}

	res, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// close connexion
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	var jsonRes []GistJSON.Response
	err = json.Unmarshal(body, &jsonRes)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	for _, val := range jsonRes {
		fmt.Printf("%s\n", val.HtmlUrl)
		fmt.Printf("(%s)\t%s\n", shortDate(val.CreatedAt), val.Description)
		fmt.Printf("\n")
	}
}

func Post(baseUrl string, accessToken string, isPublic bool, name string, content string, description string, openBrowser bool) {
	files := make(map[string]GistJSON.File)
	files[name] = GistJSON.File{Content: content}
	gist := GistJSON.Post{Desc: description, Public: isPublic, Files: files}

	// encode json
	buf, err := json.Marshal(gist)
	if err != nil {
		fmt.Printf("%s", err)
	}
	jsonBody := bytes.NewBuffer(buf)

	// post json
	postUrl := baseUrl + "gists"
	if accessToken != "" {
		postUrl = postUrl + "?access_token=" + accessToken
	}

	resp, err := http.Post(postUrl, "text/json", jsonBody)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// close connexion
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	var jsonRes GistJSON.Response
	err = json.Unmarshal(body, &jsonRes)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// copy url to clipboard
	Utils.Copy(jsonRes.HtmlUrl)

	if openBrowser {
		Utils.OpenBrowser(jsonRes.HtmlUrl)
	}

	// display result
	fmt.Printf("%s\n", jsonRes.HtmlUrl)
}

func Update(baseUrl string, accessToken string, name string, content string, gistUrl string, description string, openBrowser bool) {
	files := make(map[string]GistJSON.File)
	files[name] = GistJSON.File{Content: content}
	gist := GistJSON.Patch{Desc: description, Files: files}

	// encode json
	buf, err := json.Marshal(gist)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	jsonBody := bytes.NewBuffer(buf)

	gistId := getGistId(gistUrl)

	// post json
	postUrl := baseUrl + "gists/" + gistId
	if accessToken != "" {
		postUrl = postUrl + "?access_token=" + accessToken
	}

	req, err := http.NewRequest("PATCH", postUrl, jsonBody)
	// handle err
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// close connexion
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	var jsonErrorMessage GistJSON.MessageResponse
	err = json.Unmarshal(body, &jsonErrorMessage)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	if jsonErrorMessage.Message != "" {
		fmt.Printf("%s\n", jsonErrorMessage.Message)
		os.Exit(1)
	}

	var jsonRes GistJSON.Response
	err = json.Unmarshal(body, &jsonRes)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// copy url to clipboard
	Utils.Copy(jsonRes.HtmlUrl)

	fmt.Printf("%s\n", jsonRes.HtmlUrl)
	revisionCount := len(jsonRes.History)
	lastHistoryStatus := jsonRes.History[0].ChangeStatus
	fmt.Printf("Revision %d (%d additions & %d deletions)\n", revisionCount, lastHistoryStatus.Deletions, lastHistoryStatus.Additions)

	if openBrowser {
		Utils.OpenBrowser(jsonRes.HtmlUrl)
	}
}

func Delete(baseUrl string, accessToken string, gistUrl string) {

	gistId := getGistId(gistUrl)

	deleteUrl := baseUrl + "gists/" + gistId
	if accessToken != "" {
		deleteUrl = deleteUrl + "?access_token=" + accessToken
	}
	req, err := http.NewRequest("DELETE", deleteUrl, nil)
	// handle err
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// close connexion
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		fmt.Println("Gist deleted with success")
	} else {
		fmt.Printf("Could not find gist %s\n", gistId)
	}
}

func Download(baseUrl string, accessToken string, gistUrl string) {

	gistId := getGistId(gistUrl)

	downloadUrl := baseUrl + "gists/" + gistId
	if accessToken != "" {
		downloadUrl = downloadUrl + "?access_token=" + accessToken
	}
	res, err := http.Get(downloadUrl)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// close connexion
	defer res.Body.Close()
	if res.StatusCode != 200 {
		printErrorMessage(res)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	var jsonRes GistJSON.Response
	err = json.Unmarshal(body, &jsonRes)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	for _, file := range jsonRes.Files {
		fmt.Printf("Downloading %s\n", file.FileName)
		ioutil.WriteFile(file.FileName, []byte(file.Content), 0660)
	}
}

func shortDate(dateString string) string {
	date, err := time.Parse("2006-01-02T15:04:05Z07:00", dateString)
	if err != nil {
		fmt.Println(err)
	}
	return date.Format("2006-01-02")
}

func getGistId(urlOrId string) string {
	/*
		    accepted gist format are full url:
			     https://gist.github.com/a2a510376da5ffcb93f9
			  or just id
			     a2a510376da5ffcb93f9
			   split on '/' to retreive only id
	*/
	splitted := strings.Split(urlOrId, "/")
	return splitted[len(splitted)-1]
}

func printErrorMessage(resp *http.Response) {
	var jsonRes GistJSON.MessageResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	err = json.Unmarshal(body, &jsonRes)
	fmt.Printf("Sorry, %s\n", jsonRes.Message)
}
