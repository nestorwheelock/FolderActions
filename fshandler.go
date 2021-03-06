package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func startWatcher(dir string) {
	if verbose && !quiet {
		fmt.Println("Started watcher for", dir)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				handleEvent(dir, event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func eventToScriptPath(dir, event string) string {
	if len(dir) == 0 {
		return ""
	}
	nameDir := strings.ReplaceAll(dir, "/", "_")
	if strings.HasSuffix(nameDir, "_") {
		nameDir = nameDir[:len(nameDir)-1]
	}
	return nameDir + "_" + event + ".sh"
}

func handleEvent(dir string, event fsnotify.Event) {
	name := event.Name

	if checkfile(dir, name) {
		fmt.Printf("File %s contains insecure char! use --allow-unsafe to allow this file!", dir+name)
		return
	}

	var scriptFile string
	if event.Op&fsnotify.Create == fsnotify.Create {
		if verbose && !quiet {
			fmt.Println("create", name)
		}
		scriptFile = scriptPath + eventToScriptPath(dir, "create")
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		if verbose && !quiet {
			fmt.Println("remove:", dir, name)
		}
		scriptFile = scriptPath + eventToScriptPath(dir, "delete")
	} else {
		return
	}
	if !quiet && verbose {
		fmt.Println("to run scriptfile:", scriptFile)
	}
	err := runScript(scriptFile, name)
	if err != nil && verbose && !quiet {
		fmt.Println(err.Error())
	}
}

func runScript(scriptFile, name string) error {
	exec := exec.Command(scriptFile, name)
	out, err := exec.Output()
	if err == nil && verbose {
		s := string(out)
		if len(s) > 0 && s != "\n" {
			fmt.Println(s)
		}
	}
	return err
}

//returns true if name or dir contains critical chars
func checkfile(dir, name string) bool {
	if allowUnsafe {
		return false
	}
	ccl := []string{"!", "#", ";", "`", "\\", "|", "$", "(", ")", "{", "}", ":"}
	for _, cc := range ccl {
		if strings.Contains(dir, cc) || strings.Contains(name, cc) {
			return true
		}
	}
	return false
}
