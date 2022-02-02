package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/tools/clientcmd"
)

var version string

func findKubeConfig() (string, error) {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env, nil
	}
	path, err := homedir.Expand("~/.kube/config")
	if err != nil {
		return "", err
	}
	return path, nil
}

func chooseContext(file string) {
	k := newKubeconfig(file)
	result := k.chooseContext("Select Context")
	fmt.Printf("You choose %q\n", result)
	k.CurrentContext = result
	err := clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}

func deleteContext(file string) {
	k := newKubeconfig(file)
	result := k.chooseContext("Delete Context")

	err := clientcmd.WriteToFile(k.Config, fmt.Sprintf("%s.bak", file))
	if err != nil {
		panic(err)
	}

	k.deleteContext(result)
	k.cleanupUnusedItems()

	err = clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}

func mergeContext(file, addFile string) {
	k := newKubeconfig(file)

	err := clientcmd.WriteToFile(k.Config, fmt.Sprintf("%s.bak", file))
	if err != nil {
		panic(err)
	}

	a := newKubeconfig(addFile)
	for key, c := range a.Clusters {
		k.Clusters[key] = c
	}
	for key, u := range a.AuthInfos {
		k.AuthInfos[key] = u
	}
	for key, c := range a.Contexts {
		k.Contexts[key] = c
	}

	err = clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}

}

func main() {
	delete := false
	addFile := ""
	showVersion := false
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&delete, "d", false, "Choose context to delete")
	flag.StringVar(&addFile, "a", "", "Merge this file into kubeconfig")
	flag.Parse()

	if delete && addFile != "" {
		log.Fatalln("delete and merge is not allowed")
	}
	file, err := findKubeConfig()
	if err != nil {
		panic(err)
	}

	switch {
	case showVersion:
		fmt.Println(version)
	case delete:
		deleteContext(file)
	case addFile != "":
		mergeContext(file, addFile)
	default:
		chooseContext(file)
	}
}
