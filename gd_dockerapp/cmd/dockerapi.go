package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const (
	// ATLAS_HOST     = "http://192.168.3.18:8088"
	// ATLAS_IMG_PATH = "/gddockerapp/cmd/muchener-testcommitcp-v1.tar"
	// ATLAS_IMG_NAME = "muchener/testcommitcp"

	//CONF_FILE_PATH = "/gddockerapp/cmd/res/gddocker.conf"
	CONF_FILE_PATH = "/gddockerapp/cmd/res/gddocker.conf"
	HOST           = "host"
	PORT           = "port"
	IMAGEDIR       = "imagedir"
	IMAGENAME      = "imagename"
)

type GdDocker struct {
	AtlasAddress       string
	AtlasImageDir      string
	AtlasImageFullName []string
	AtlasImageName     []string
	AtlasConIDMap      map[string]string
}

var once sync.Once
var gd *GdDocker

func NewGddocker() *GdDocker {
	once.Do(func() {
		gd = new(GdDocker)
		gd.AtlasConIDMap = make(map[string]string)
	})
	return gd
}

func (gd *GdDocker) updateornot() bool {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
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
	var loadflag bool
	for _, im := range gd.AtlasImageName {
		loadflag = true
		for _, image := range images {
			fmt.Println(image.Containers, image.ID, image.RepoTags, gd.AtlasImageName)
			if image.RepoTags[0] == im {
				loadflag = false
			}
		}
		if loadflag {
			gd.AtlasImageFullName = append(gd.AtlasImageFullName, gd.AtlasImageDir+im+".tar")
		}
	}
	fmt.Printf("new image to load:%s", gd.AtlasImageFullName)
	if cap(gd.AtlasImageFullName) > 0 {
		return true
	} else {
		return false
	}
}

//1.启动APP后，自动对比版本和load镜像包
func (gd *GdDocker) loadImage() {

	if gd.updateornot() {

		ctx := context.Background()
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
		if err != nil {
			panic(err)
		}
		fmt.Println(gd.AtlasImageFullName)
		// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
		// if err != nil {
		// 	panic(err)
		// }
		// io.Copy(os.Stdout, reader)
		for _, imfullname := range gd.AtlasImageFullName {
			file, err := os.Open(imfullname)
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
			file.Close()
		}
	}

}

func (gd *GdDocker) conInstall(containerName string, imageName string) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		// Cmd:   []string{"echo", "hello world2"},
	}, nil, nil, nil, containerName) //镜像名称作为容器名称
	if err != nil {
		code = 400
		msg = err.Error()
		//panic(err)
	} else {
		code = 200
		msg = containerName + " install success"
		gd.AtlasConIDMap[containerName] = resp.ID
	}
	fmt.Printf("installed container:%s", gd.AtlasConIDMap)
	return code, msg

}
func (gd *GdDocker) conStart(containerName string) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	//start容器
	containerID, err := gd.getIDbyContainerName(containerName)
	if err != nil {
		code = 400
		msg = err.Error()
		return code, msg
	}
	fmt.Printf("conStart:%s - %s", containerName, containerID)
	err = cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = containerName + " start success"
	}
	return code, msg
}
func (gd *GdDocker) conStop(containerName string) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	//start容器
	containerID, err := gd.getIDbyContainerName(containerName)
	if err != nil {
		code = 400
		msg = err.Error()
		return code, msg
	}
	fmt.Printf("conStop:%s - %s", containerName, containerID)
	timeout := 100 * time.Second
	err = cli.ContainerStop(ctx, containerID, &timeout)
	if err != nil {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = containerName + " stop success"
	}
	return code, msg
}
func (gd *GdDocker) conRemove(containerName string) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	//start容器
	containerID, err := gd.getIDbyContainerName(containerName)
	if err != nil {
		code = 400
		msg = err.Error()
		return code, msg
	}
	fmt.Printf("conRemove:%s - %s", containerName, containerID)

	err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = containerName + " remove success"
	}
	return code, msg
}

func (gd *GdDocker) version(w http.ResponseWriter, r *http.Request) {

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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
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
			if strings.Contains(tag, gd.AtlasImageName[0]) { //版本不同
				// if tag == "muchener/testcommitcp:v1" {
				fmt.Println(w, tag)
				break
			}
		}
	}

	fmt.Fprintf(w, "version!") //这个写入到w的是输出到客户端的
}
func (gd *GdDocker) imageUpgrade(w http.ResponseWriter, r *http.Request) {
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	file, err := os.Open(gd.AtlasImageFullName[0])
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
		// Cmd:   []string{"echo", "hello world2"},
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
func (gd *GdDocker) containerState(w http.ResponseWriter, r *http.Request) {

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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	// containerID = gd.getIDbyContainerName(containerName)

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
func (gd *GdDocker) getIDbyImageName(imagename string) string {
	imageID := "imageid"

	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
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

func (gd *GdDocker) removeImage(w http.ResponseWriter, r *http.Request) {
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	imageID = gd.getIDbyImageName(imageName)
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

func (gd *GdDocker) containersList(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
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
func (gd *GdDocker) getIDbyContainerName(containername string) (containerID string, err error) {

	//获取容器名对应的容器id
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	options := types.ContainerListOptions{
		All: true,
	}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		// panic(err)
		return "_", err
	}
	for _, container := range containers {
		fmt.Printf("getIDbyContainerName:%s,%s,%s", container.Names, container.ID, containername)
		//fmt.Println(container.Names[0]) //[/determined_haslett]
		if container.Names[0] == "/"+containername {
			containerID = container.ID
			break
		}
	}
	return containerID, err
}
func (gd *GdDocker) containersDelete(w http.ResponseWriter, r *http.Request) {
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	// containerID = gd.getIDbyContainerName(containerName)
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

func (gd *GdDocker) containerRestart(w http.ResponseWriter, r *http.Request) {
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	// containerID = gd.getIDbyContainerName(containerName)
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

//读取key=value类型的配置文件
func (gd *GdDocker) InitConfig(path string) {
	config := make(map[string]string)

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		s := strings.TrimSpace(string(b))
		index := strings.Index(s, "=")
		if index < 0 {
			continue
		}
		key := strings.TrimSpace(s[:index])
		if len(key) == 0 {
			continue
		}
		value := strings.TrimSpace(s[index+1:])
		if len(value) == 0 {
			continue
		}
		config[key] = value
	}
	gd.AtlasAddress = "http://" + config[HOST] + ":" + config[PORT]
	gd.AtlasImageDir = config[IMAGEDIR]
	gd.AtlasImageName = strings.Fields(config[IMAGENAME])

}
func (gd *GdDocker) init() {
	//读取并获取镜像名称
	//解析文件
	gd.InitConfig(CONF_FILE_PATH)

	fmt.Println(gd)
	//使用文件
	// files, _ := ioutil.ReadDir(gd.AtlasImageDir)
	// for i, f := range files {
	// 	if strings.Contains(f.Name(), ".tar") {
	// 		gd.AtlasImageFullName[i] = gd.AtlasImageDir + f.Name()
	// 		fmt.Println(gd.AtlasImageFullName[i])
	// 	}
	// }
}
func (gd *GdDocker) imgUpdate() error {

	var containerName string

	containerName = strings.ReplaceAll(gd.AtlasImageName[0], "/", "-")
	containerName = strings.ReplaceAll(containerName, ":", "-")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	//获取镜像版本，判断是否需要升级
	if gd.updateornot() {
		file, err := os.Open(gd.AtlasImageFullName[0])
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
			Image: gd.AtlasImageName[0],
			//Cmd:   []string{"echo", "hello world2"},
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
	} else {
		fmt.Println("Current version is updated!")
	}

	return err
}
