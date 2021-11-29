// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func version(w http.ResponseWriter, r *http.Request) {

	// r.ParseForm()       //解析参数，默认是不会解析的
	// fmt.Println(r.Form) //这些信息是输出到服务器端的打印信息
	// fmt.Println("path", r.URL.Path)
	// fmt.Println("scheme", r.URL.Scheme)
	// fmt.Println(r.Form["url_long"])
	// for k, v := range r.Form {
	// 	fmt.Println("key:", k)
	// 	fmt.Println("val:", strings.Join(v, ""))
	// }
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.RepoTags[0])
		for _, tag := range image.RepoTags {
			if strings.Contains(tag, "muchener/testcommitcp") { //版本不同
				// if tag == "muchener/testcommitcp:v1" {
				fmt.Fprintf(w, tag)
				break
			}
		}
	}

	fmt.Fprintf(w, "version!") //这个写入到w的是输出到客户端的
}
func imageUpgrade(w http.ResponseWriter, r *http.Request) {
	var imageName string
	var containerName string
	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		if k == "imagename" {
			imageName = v[0]
			break
		}
	}

	containerName = strings.ReplaceAll(imageName, "/", "-")
	containerName = strings.ReplaceAll(containerName, ":", "-")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}

	file, err := os.Open("./img/muchener-testcommitcp:v1.tar")
	if err != nil {
		fmt.Println("err=", err)
	}
	imageLoadResponse, err := cli.ImageLoad(ctx, file, true)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(imageLoadResponse.Body)
	if err != nil {
		fmt.Println(" load err=", err)
	}
	fmt.Println(string(body))

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"echo", "hello world2"},
	}, nil, nil, nil, containerName) //镜像名称作为容器名称
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	fmt.Fprintf(w, "image upgrade success!") //这个写入到w的是输出到客户端的
}
func containerState(w http.ResponseWriter, r *http.Request) {

	var containerName string
	containerID := "containerid"

	r.ParseForm()
	for k, v := range r.Form {
		if k == "containername" {
			containerName = v[0]
			break
		}
	}
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}

	containerID = getIDbyContainerName(containerName)

	if containerID == "containerid" {
		fmt.Fprintf(w, "No container named %s\n", containerName)
	} else {
		//查看容器状态
		resp, err := cli.ContainerStats(ctx, containerID, false)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "%s", content) //[/muchener-testcommitcp-v1] cffd3b0a35042f16eed861faaa671d0fbdfb53918f5c06e004668f13b2c69b34 muchener-testcommitcp-v1
		fmt.Fprintf(w, "image containerState success!")
	}
}
func getIDbyImageName(imagename string) string {
	imageID := "imageid"

	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}

	filters := filters.NewArgs()
	//filters.Add("label", "muchener/testcommitcp")
	// filters.Add("dangling", "true")
	options := types.ImageListOptions{
		Filters: filters,
	}

	images, err := cli.ImageList(ctx, options)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(images))
	// if len(images) != 2 {
	// 	panic("expected 2 images, got %v", images)
	// }

	for _, image := range images {
		fmt.Println(image.Containers, image.ID, image.RepoTags, imagename)
		if image.RepoTags[0] == imagename {
			imageID = image.ID
			break
		}
	}
	return imageID
}
func loadImage(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}

	// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// io.Copy(os.Stdout, reader)
	file, err := os.Open("./img/muchener-testcommitcp:v1.tar")
	if err != nil {
		fmt.Println("err=", err)
	}
	imageLoadResponse, err := cli.ImageLoad(ctx, file, true)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(imageLoadResponse.Body)
	if err != nil {
		fmt.Println(" load err=", err)
	}
	fmt.Println(string(body))
	fmt.Fprintf(w, "image load success!") //这个写入到w的是输出到客户端的
}
func removeImage(w http.ResponseWriter, r *http.Request) {
	var imageID string
	var imageName string
	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		if k == "imagename" {
			imageName = v[0]
			break
		}
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	imageID = getIDbyImageName(imageName)
	imageDeletes, err := cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: false,
	})
	if err != nil {
		panic(err)
	}
	if len(imageDeletes) != 2 { //todo
		fmt.Printf("expected 2 deleted images, got %v", imageDeletes)
	}
	fmt.Fprintf(w, "image remove success!") //这个写入到w的是输出到客户端的
}

func containersList(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	options := types.ContainerListOptions{
		All: true,
	}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		fmt.Println(container.Names)
		fmt.Fprintf(w, container.Names[0])
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "containersList\n")
}
func getIDbyContainerName(containername string) string {
	containerID := "containerid"
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	options := types.ContainerListOptions{
		All: true,
	}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		fmt.Println(container.Names, container.ID, containername)
		//fmt.Println(container.Names[0]) //[/determined_haslett]
		if container.Names[0] == "/"+containername {
			containerID = container.ID
			break
		}
	}
	return containerID
}
func containersDelete(w http.ResponseWriter, r *http.Request) {
	var containerName string
	containerID := "containerid"

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		if k == "containername" {
			containerName = v[0]
			break
		}
	}
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}

	containerID = getIDbyContainerName(containerName)
	if containerID == "containerid" {
		fmt.Fprintf(w, "No container named %s\n", containerName)
	} else {
		//删除容器
		err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "containers %s Delete success!\n", containerName)
	}
}
func containerStart(w http.ResponseWriter, r *http.Request) {
	var containerName string
	containerID := "containerid"

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		// if strings.Contains(k, "containername") {
		if k == "containername" {
			containerName = v[0]
			break
		}
	}
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	containerID = getIDbyContainerName(containerName)
	if containerID == "containerid" {
		fmt.Fprintf(w, "No container named %s\n", containerName)
	} else {
		//start容器
		err = cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "containers %s start success!\n", containerName)
	}
}
func containerStop(w http.ResponseWriter, r *http.Request) {
	var containerName string
	containerID := "containerid"

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		// if strings.Contains(k, "containername") {
		if k == "containername" {
			containerName = v[0]
			break
		}
	}
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	containerID = getIDbyContainerName(containerName)
	if containerID == "containerid" {
		fmt.Fprintf(w, "No container named %s\n", containerName)
	} else {
		//start容器
		timeout := 100 * time.Second
		err = cli.ContainerStop(ctx, containerID, &timeout)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "containers %s stop success!\n", containerName)
	}
}
func containerRestart(w http.ResponseWriter, r *http.Request) {
	var containerName string
	containerID := "containerid"

	r.ParseForm()
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
		// if strings.Contains(k, "containername") {
		if k == "containername" {
			containerName = v[0]
			break
		}
	}
	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
	if err != nil {
		panic(err)
	}
	containerID = getIDbyContainerName(containerName)
	if containerID == "containerid" {
		fmt.Fprintf(w, "No container named %s\n", containerName)
	} else {
		//start容器
		timeout := 100 * time.Second
		err = cli.ContainerRestart(ctx, containerID, &timeout)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "containers %s restart success!\n", containerName)
	}
}
func main() {

	http.HandleFunc("/images/version", version)      //设置访问的路由
	http.HandleFunc("/images/upgrade", imageUpgrade) //load-create-start:http://192.168.198.128:9090/images/upgrade
	http.HandleFunc("/images/state", containerState)

	http.HandleFunc("/images/delete", removeImage) //http://192.168.198.128:9090/images/delete
	http.HandleFunc("/images/load", loadImage)     //http://192.168.198.128:9090/images/load

	http.HandleFunc("/container/list", containersList)      //http://192.168.198.128:9090/container/list
	http.HandleFunc("/container/delete", containersDelete)  //http://192.168.198.128:9090/container/delete?containername=determined_haslett
	http.HandleFunc("/container/start", containerStart)     //http://192.168.198.128:9090/container/start?containername=determined_haslett
	http.HandleFunc("/container/stop", containerStop)       //http://192.168.198.128:9090/container/stop?containername=determined_haslett
	http.HandleFunc("/container/restart", containerRestart) //http://192.168.198.128:9090/container/stop?containername=determined_haslett

	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"io/ioutil"
// 	"net/http"
// 	"os"

// 	"github.com/docker/docker/api/types"
// 	"github.com/docker/docker/api/types/container"
// 	"github.com/docker/docker/client"
// 	"github.com/docker/docker/pkg/stdcopy"
// )

// // struct RepoTags{

// // }
// func postFile() {
// 	//这是一个Post 参数会被返回的地址
// 	strinUrl := "http://192.168.3.18:8080/aaa"
// 	byte, err := ioutil.ReadFile("post.txt")
// 	resopne, err := http.Post(strinUrl, "multipart/form-data", bytes.NewReader(byte)) //二进制文件
// 	if err != nil {
// 		fmt.Println("err=", err)
// 	}
// 	defer func() {
// 		resopne.Body.Close()
// 		fmt.Println("finish")
// 	}()
// 	body, err := ioutil.ReadAll(resopne.Body)
// 	if err != nil {
// 		fmt.Println(" post err=", err)
// 	}
// 	fmt.Println(string(body))
// }
// func importfile() {

// 	file, err := os.Open("/root/test.tar")
// 	if err != nil {
// 		fmt.Println("err=", err)
// 	}
// 	resp, err := http.Post("http://192.168.3.18:8088/images/load", "application/json", file)
// 	if err != nil {
// 		fmt.Println("err=", err)
// 	}
// 	defer func() {
// 		resp.Body.Close()
// 		fmt.Println("finish")
// 	}()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println(" post err=", err)
// 	}
// 	fmt.Println(string(body))
// }

// // TODO: handle errors

// // func main() {

// // 	importfile()
// // }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost("http://192.168.3.18:8088"))
// 	if err != nil {
// 		panic(err)
// 	}

// 	// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// io.Copy(os.Stdout, reader)
// 	file, err := os.Open("/root/test.tar")
// 	if err != nil {
// 		fmt.Println("err=", err)
// 	}
// 	imageLoadResponse, err := cli.ImageLoad(ctx, file, true)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := ioutil.ReadAll(imageLoadResponse.Body)
// 	if err != nil {
// 		fmt.Println(" load err=", err)
// 	}
// 	fmt.Println(string(body))

// 	resp, err := cli.ContainerCreate(ctx, &container.Config{
// 		Image: "muchener/testcommitcp:v1", //muchener/testcommitcp:v1
// 		Cmd:   []string{"echo", "hello world"},
// 	}, nil, nil, nil, "gddockerapp1")
// 	if err != nil {
// 		panic(err)
// 	}

// 	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
// 		panic(err)
// 	}

// 	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
// 	select {
// 	case err := <-errCh:
// 		if err != nil {
// 			panic(err)
// 		}
// 	case <-statusCh:
// 	}

// 	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
// 	if err != nil {
// 		panic(err)
// 	}

// 	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
// }
