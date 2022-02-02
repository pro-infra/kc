package main

import (
	"log"

	"github.com/manifoldco/promptui"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type kubeconfig struct {
	api.Config
}

func newKubeconfig(file string) *kubeconfig {
	kubeConfig, err := clientcmd.LoadFromFile(file)
	if err != nil {
		panic(err)
	}
	var k kubeconfig
	k.Config = *kubeConfig
	return &k
}

func (k *kubeconfig) chooseContext(label string) string {
	list := []string{}
	for n := range k.Contexts {
		list = append(list, n)
	}
	prompt := promptui.Select{
		Label: label,
		Items: list,
		Size:  10,
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return result
}

func (k *kubeconfig) deleteContext(context string) {
	if len(k.Contexts) <= 1 {
		log.Fatalln("Only one Context - Nothing deleted - Delete the hole file")
	}
	delete(k.Contexts, context)
	if _, ok := k.Contexts[k.CurrentContext]; !ok {
		for n := range k.Contexts {
			log.Printf("current context was deleted - used %s", n)
			k.CurrentContext = n
			return
		}
	}
}

func (k *kubeconfig) cleanupUnusedItems() {
	clusters := map[string]bool{}
	for c := range k.Clusters {
		clusters[c] = true
	}
	users := map[string]bool{}
	for u := range k.AuthInfos {
		users[u] = true
	}
	for _, c := range k.Contexts {
		delete(clusters, c.Cluster)
		delete(users, c.AuthInfo)
	}
	for c := range clusters {
		delete(k.Clusters, c)
	}
	for u := range users {
		delete(k.AuthInfos, u)
	}
}
