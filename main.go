package main

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	homedir "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

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
func getContexts(kubeConfig *api.Config) []string {
	list := []string{}
	for n := range kubeConfig.Contexts {
		list = append(list, n)
	}
	return list
}

func main() {
	k, err := findKubeConfig()
	if err != nil {
		panic(err)
	}
	kubeConfig, err := clientcmd.LoadFromFile(k)
	if err != nil {
		panic(err)
	}

	prompt := promptui.Select{
		Label: "Select Context",
		Items: getContexts(kubeConfig),
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	fmt.Printf("You choose %q\n", result)
	kubeConfig.CurrentContext = result
	err = clientcmd.WriteToFile(*kubeConfig, k)
	if err != nil {
		panic(err)
	}
}
