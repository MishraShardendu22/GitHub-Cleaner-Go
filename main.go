package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Repo struct {
	Name string `json:"name"`
}

func main() {
	username := "MishraShardendu22"
	url := "https://api.github.com/users/" + username + "/repos?per_page=100"

	repos := getRepos(url)
	for _, repo := range repos {
		Clone(repo)
	}
}

func getRepos(url string) []string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var repos []Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		log.Fatal(err)
	}

	var names []string
	for _, r := range repos {
		names = append(names, r.Name)
	}
	return names
}

func Clone(repo string) {
	repoURL := "git@github.com:MishraShardendu22/" + repo + ".git"
	cmd := exec.Command("git", "clone", repoURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	if err := os.Chdir(repo); err != nil {
		fmt.Println("Chdir failed:", err)
		os.RemoveAll(repo)
		return
	}
	defer func() {
		os.Chdir("..")
		os.RemoveAll(repo)
	}()

	Cleaner()
}

func Cleaner() {
	start, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Starting in:", start)
	DeepSearchAndClean(start)
}

func DeepSearchAndClean(folder string) {
	files := Folder(folder, false)
	dirs := Folder(folder, true)

	if Contains(files, "package.json") {
		CleanThis(folder)
		return
	}

	for _, d := range dirs {
		DeepSearchAndClean(filepath.Join(folder, d))
	}
}

func CleanThis(folder string) {
	content, err := os.ReadFile(filepath.Join(folder, "package.json"))
	if err != nil {
		return
	}
	pkg := string(content)
	if !strings.Contains(pkg, "react") || !strings.Contains(pkg, "react-dom") {
		return
	}

	uiDir, ok := findUIDir(folder)
	if !ok {
		return
	}
	fmt.Println("Cleaning UI in:", uiDir)

	used := map[string]bool{}
	exts := map[string]bool{".ts": true, ".tsx": true, ".js": true, ".jsx": true}

	filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !exts[filepath.Ext(path)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range regexp.MustCompile(`[./@"]components/ui/([A-Za-z0-9_-]+)`).FindAllStringSubmatch(string(data), -1) {
			used[strings.ToLower(m[1])] = true
		}
		return nil
	})

	entries, err := os.ReadDir(uiDir)
	if err != nil {
		fmt.Println("Failed to read ui directory:", err)
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
		if !used[base] {
			path := filepath.Join(uiDir, name)
			fmt.Println("Deleting unused:", path)
			os.RemoveAll(path)
		}
	}

	build := exec.Command("sh", "-c", "npm install --legacy-peer-deps && npm run build")
	build.Dir = folder
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	build.Run()

	git := exec.Command("sh", "-c", "git cm 'auto: cleanup ui and build'")
	git.Dir = folder
	git.Stdout = os.Stdout
	git.Stderr = os.Stderr
	git.Run()
}

func findUIDir(root string) (string, bool) {
	var uiDir string
	found := false
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if found || err != nil || !d.IsDir() || filepath.Base(path) != "ui" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == "components" {
			uiDir = path
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return uiDir, found
}

func Contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func Folder(root string, wantDir bool) []string {
	entries, _ := os.ReadDir(root)
	var items []string
	for _, e := range entries {
		if wantDir && e.IsDir() {
			items = append(items, e.Name())
		}
		if !wantDir && !e.IsDir() {
			items = append(items, e.Name())
		}
	}
	return items
}
