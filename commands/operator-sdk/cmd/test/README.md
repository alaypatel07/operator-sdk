### Steps to check ansible tests

1. Clone the repo
2. Issue the command `make all` to compile the code
3. Issue the command 
    ```
        operator-sdk test local  <path-to-directory>
        --namespaced-manifest $(pwd)/example/deploy/namespace-init.yaml
        --global-manifest $(pwd)/example/deploy/crd.yaml
        --type ansible
    ```
    where `<path-to-directory>` is the path to test directory

4. The test directory should be organized as follows:

    ```

    test
    ├── test-name
        ├── 00
        │   ├── assert.yaml
        │   └── create.yaml
        ├── 01
        │   ├── assert.yaml
        │   └── udpate.yaml
        ├── 03
            ├── assert.yaml
            └── delete.yaml

    ```
    
    example [here](https://github.com/alaypatel07/ansible-operator/tree/e2e-tests/example/test/test-example)
    
5. To test the [etcd-ansible-operator](https://github.com/water-hole/etcd-ansible-operator), use the following commands:

    ```
        $mkdir -p $GOPATH/src/github.com/water-hole && cd $GOPATH/src/github.com/water-hole && git clone https://github.com/water-hole/etcd-ansible-operator && cd etcd-ansible-operator
        $ operator-sdk test local  ./test --namespaced-manifest $(pwd)/deploy/namespace-init.yaml --global-manifest $(pwd)/deploy/crd.yaml --type ansible 
    ```
    
    Tip: use `$watch kubectl get pods --all-namespaces` to watch the pods come up and die during the running the tests