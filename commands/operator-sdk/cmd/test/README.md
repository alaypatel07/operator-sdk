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
    ├── test-example
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