package cmdtest

import (
	"testing"
	"time"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"io"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"context"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"errors"
	"path/filepath"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"sort"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval = time.Second * 5
	timeout       = time.Second * 30
	testDir       string
)

type Assert struct {
	Resource map[string]string      `json:"resource"`
	Result   map[string]interface{} `json:"result"`
}

func TestExample(t *testing.T) {
	testDir = os.Getenv("TESTDIR")
	if testDir == "" {
		testDir = "."
	}
	// run subtests
	t.Run("ansible-group", func(t *testing.T) {
		t.Run("Cluster", AnsibleCluster)
	})
}

func readAsserts(reader io.Reader) ([]Assert, error) {
	d := yaml.NewYAMLOrJSONDecoder(reader, 65535)
	var asserts []Assert
	err := d.Decode(&asserts)
	return asserts, err
}

func readCrs(reader io.Reader, namespace string) ([]*unstructured.Unstructured, error) {
	crs := make([]*unstructured.Unstructured, 0)
	d := yaml.NewYAMLOrJSONDecoder(reader, 65535)
	var cr unstructured.Unstructured;
	for {
		err := d.Decode(&cr)
		if err == nil {
			cr.SetNamespace(namespace)
			crs = append(crs, &cr)
		}
		if err == io.EOF {
			return crs, nil
		}
		if err != nil {
			return crs, err
		}
	}
}

func getTestDirs(td string) ([]string, error) {
	testDirs := make([]string, 0)
	err := filepath.Walk(td, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && path[0:1] != "." {
			testDirs = append(testDirs, path)
		}
		return nil
	})
	return testDirs, err
}

func AnsibleCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Logf("could not get namespace: %v", err)
	}
	defer ctx.Cleanup(t)

	fmt.Println("start")
	err = ctx.InitializeClusterResources()
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	fmt.Println("Initialized")
	t.Log("Initialized cluster resources")
	fmt.Println("namespaced")
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "ansible-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Deployment successfull")

	fmt.Println("Registerging appGVK")

	appGVK := schema.GroupVersion{
		Group:   "app.example.com",
		Version: "v1alpha1",
	}

	framework.AddToFrameworkScheme(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(appGVK, &unstructured.Unstructured{},
			&unstructured.UnstructuredList{})
		metav1.AddToGroupVersion(scheme, appGVK)
		return nil
	}, &unstructured.UnstructuredList{
		Items: nil,
	})

	testDirs, err := getTestDirs(testDir + "/test-example")

	sort.Strings(testDirs)


	for _, dir := range testDirs {

		cf, err := os.Open(dir + "/create.yaml")
		if err == nil {
			c, err := readCrs(cf, namespace)
			if err != nil {
				panic(err)
			}
			ctx.AddFinalizerFn(func() error {
				for _, cr := range c {
					framework.Global.DynamicClient.Delete(context.TODO(), cr)
				}
				return nil
			})
			for _, cr := range c {
				err = framework.Global.DynamicClient.Create(context.TODO(), cr)
				if err != nil {
					t.Logf("error creating crs %+v\n", err)
					t.Fail()
				}
			}
		}
		uf, err := os.Open(dir + "/update.yaml")
		if err == nil {
			u, err := readCrs(uf, namespace)
			if err != nil {
				panic(err)
			}
			ctx.AddFinalizerFn(func() error {
				for _, cr := range u {
					framework.Global.DynamicClient.Delete(context.TODO(), cr)
				}
				return nil
			})
			for _, cr := range u {
				ok := types.NamespacedName{cr.GetNamespace(), cr.GetName()}
				var c unstructured.Unstructured
				err := framework.Global.DynamicClient.Get(context.TODO(), ok, &c)
				if err != nil {
					t.Logf("error getting cr for update %+v\n", err)
					t.Fail()
				}
				cr.SetResourceVersion(c.GetResourceVersion())
				err = framework.Global.DynamicClient.Update(context.TODO(), cr)
				if err != nil {
					t.Logf("error updating cr %+v\n", err)
					t.Fail()
				}
			}
		}
		df, err := os.Open(dir + "/delete.yaml")
		if err == nil {
			d, err := readCrs(df, namespace)
			if err != nil {
				panic(err)
			}
			for _, cr := range d {
				err = framework.Global.DynamicClient.Delete(context.TODO(), cr)
				if err != nil {
					t.Logf("error deleting crs %+v\n", err)
					t.Fail()
				}
			}
		}

		r, err := os.Open(dir + "/assert.yaml")
		if err != nil {
			fmt.Printf("error reading assert.yaml %s\n", err.Error())
			t.Logf("error reading assert.yaml %s\n", err.Error())
		}

		asserts, err := readAsserts(r)
		if err != nil {
			fmt.Printf("error loading asserts %s\n", err.Error())
			t.Logf("error loading asserts %s\n", err.Error())
		}

		resources := make([]*unstructured.Unstructured, 0)
		results := make([]map[string]interface{}, 0)

		fmt.Printf("Printing asserts\n")

		for _, assert := range asserts {
			//fmt.Printf("%d, %+v\n", i, assert)
			u := unstructured.Unstructured{}
			gv, err := schema.ParseGroupVersion(assert.Resource["apiVersion"])
			if err != nil {
				panic("error converting gvk")
			}
			gvk := gv.WithKind(assert.Resource["kind"])
			u.SetGroupVersionKind(gvk)
			resources = append(resources, &u)
			results = append(results, assert.Result)
		}
		fmt.Printf("Waiting for asserts\n")

		err = WaitForResources(t, resources, results, namespace, retryInterval, timeout)
		if err != nil {
			t.Logf("error matching asserts %s\n", err)
			t.Fail()
		}
	}
}

func WaitForResources(t *testing.T, resources []*unstructured.Unstructured, results []map[string]interface{}, namespace string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		for i, r := range resources {

			u := &unstructured.Unstructured{}

			fmt.Printf("GVK is %+v %v\n", r.GroupVersionKind(), i)
			lo := crclient.InNamespace(namespace)

			u.SetGroupVersionKind(r.GroupVersionKind())
			//u.SetNamespace(r.GetNamespace())
			//u.SetName(r.GetName())

			err := framework.Global.DynamicClient.List(context.TODO(), lo, u)
			if err != nil {
				fmt.Printf("err getting list %s\n", err.Error())
			}
			counter := 0
			err = u.EachListItem(func(object runtime.Object) error {
				counter++
				return nil
			})
			if err != nil {
				panic("error calling each item")
			}

			if rc, ok := results[i]["number"].(float64); ok {
				if int(rc) != counter {
					fmt.Printf("counter is not equal expected counter %v\n", counter)
					t.Logf("waiting for the counter to equal expected number")
					return false, nil
				} else {
					t.Logf("counter is equal to expected counter")
					return true, nil
				}
			} else {
				fmt.Printf("cannot convert result.number to float64")
				return false, errors.New("cannot convert result.number to float64")
			}
		}
		return false, nil
	})
	return err
}
