# kc

A small program to quickly change the Kubernetes context from the command line.

## Usage

kc respects the environment variable KUBECONFIG. If it is not set, it uses `~/.kube/config`. 

### Change Context

To simply change the context, use the following command

```sh
kc
```

### Deleting a Context

This function deletes a kubeconfig context. After the context has been deleted, all cluster 
and user/auth information are scanned, if there are entries without reference in the context, 
these will be deleted. This is independent of the actual deleted context.
A backup file is also created. This is the path with the extension .bak. 

```sh
kc -d
```

### Merging a other Context file into target Context file.

Sometimes you want to merge another kubeconfig file into the kubeconfig you are using. 
This function allows to add a file. The added file may overwrite users or clusters with 
the same name. Also here a backup file is created before. 

```sh
kc -a NEW_KUBE_CONFIG_FILE
```
