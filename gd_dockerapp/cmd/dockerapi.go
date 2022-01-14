package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	DEVICEID       = "deviceid"
)

type GdDocker struct {
	//从配置文件读取的参数
	AtlasAddress        string
	AtlasImageDir       string
	AtlasImageFullNames []string
	AtlasImageNames     []string
	AtlasDeviceId       string
	//从报文中读取的参数
	AtlasConIDMap map[string]string
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
	for _, im := range gd.AtlasImageNames {
		loadflag = true
		for _, image := range images {
			fmt.Println(image.Containers, image.ID, image.RepoTags, gd.AtlasImageNames)
			if image.RepoTags[0] == im {
				loadflag = false
			}
		}
		if loadflag {
			gd.AtlasImageFullNames = append(gd.AtlasImageFullNames, gd.AtlasImageDir+im+".tar")
		}
	}
	fmt.Printf("new image to load:%s", gd.AtlasImageFullNames)
	if cap(gd.AtlasImageFullNames) > 0 {
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
		fmt.Println(gd.AtlasImageFullNames)
		// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
		// if err != nil {
		// 	panic(err)
		// }
		// io.Copy(os.Stdout, reader)
		for _, imfullname := range gd.AtlasImageFullNames {
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
	if err != nil || containerID == "_" {
		code = 400
		if containerID == "_" {
			msg = fmt.Sprintf("container %s don't exit", containerName)
		} else {
			msg = err.Error()
		}
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
	if err != nil || containerID == "_" {
		code = 400
		if containerID == "_" {
			msg = fmt.Sprintf("container %s don't exit", containerName)
		} else {
			msg = err.Error()
		}
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
	if err != nil || containerID == "_" {
		code = 400
		if containerID == "_" {
			msg = fmt.Sprintf("container %s don't exit", containerName)
		} else {
			msg = err.Error()
		}
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
func (gd *GdDocker) conStatus(containerName string, param *ServiceReplyParam) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	//查看容器名对应的id，没有就返回
	containerID, err := gd.getIDbyContainerName(containerName)
	if err != nil || containerID == "_" {
		code = 400
		if containerID == "_" {
			msg = fmt.Sprintf("container %s don't exit", containerName)
		} else {
			msg = err.Error()
		}
		return code, msg
	}
	// fmt.Printf("conStatus:%s - %s", containerName, containerID)
	/////////////////////////////////方法1
	resp, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content))

	m := new(STATUSREPLY)
	err = json.Unmarshal(content, &m)
	if err != nil {
		fmt.Println("Umarshal failed:", err)
		return
	}
	// fmt.Println(m.Read)

	r, err1 := cli.ContainerInspect(ctx, containerID)
	if err1 != nil {
		panic(err)
	}
	//将stats和inspect信息赋值给状态参数
	tt0 := r.State.StartedAt[0:19]
	tt1 := strings.Replace(tt0, "T", " ", 1)
	Time1, _ := time.Parse("2006-01-02 15:04:05", tt1)
	TimeNow := time.Now()
	lifelong := TimeNow.Sub(Time1)
	var cpup, memp int

	if r.State.Status == "running" { //exited,running
		cpuTotalUsage := m.Cpu_stats.System_cpu_usage / 10000000
		preCpuTotalUsage := m.Precpu_stats.System_cpu_usage / 10000000

		cpupraw := m.Cpu_stats.Online_cpus * (m.Cpu_stats.Cpu_usage.Total_usage - m.Precpu_stats.Cpu_usage.Total_usage) / (cpuTotalUsage - preCpuTotalUsage) / 100000
		cpup = (int)(cpupraw)
		fmt.Println(cpup)
		mempraw := 100.0 * (m.Memory_stats.Usage - m.Memory_stats.Stats.Cache) / m.Memory_stats.Limit
		memp = (int)(mempraw)
		fmt.Println(memp)
	} else {
		cpup = 0
		memp = 0
	}

	param.Cmd = CMD_CON_STATUS
	constatusdata := ConStatusReply{
		Container: containerName,
		// Version:   " ",
		State:    r.State.Status,
		CpuRate:  cpup,
		Memory:   memp,
		Disk:     0, //未配置
		Ip:       r.NetworkSettings.IPAddress,
		Created:  r.Created,
		Started:  r.State.StartedAt,
		LifeTime: (int64)(lifelong.Hours()),
		Image:    r.Config.Image,
	}

	param.Paras = constatusdata
	fmt.Println(param.Paras.(ConStatusReply))
	// info, err := cli.Info(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(info)
	if err != nil {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = containerName + " get status success"
	}
	return code, msg
}
func getstatus(cli *client.Client, ctx context.Context, name string, id string, csr *ConStatusReply) (code int, msg string) {
	/////////////////////////////////方法1
	resp, err := cli.ContainerStats(ctx, id, false)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(content))

	m := new(STATUSREPLY)
	err = json.Unmarshal(content, &m)
	if err != nil {
		fmt.Println("Umarshal failed:", err)
		return
	}
	// fmt.Println(m.Read)

	r, err1 := cli.ContainerInspect(ctx, id)
	if err1 != nil {
		panic(err)
	}
	//将stats和inspect信息赋值给状态参数
	tt0 := r.State.StartedAt[0:19]
	tt1 := strings.Replace(tt0, "T", " ", 1)
	Time1, _ := time.Parse("2006-01-02 15:04:05", tt1)
	TimeNow := time.Now()
	lifelong := TimeNow.Sub(Time1)
	var cpup, memp int

	if r.State.Status == "running" { //exited,running
		cpuTotalUsage := m.Cpu_stats.System_cpu_usage / 10000000
		preCpuTotalUsage := m.Precpu_stats.System_cpu_usage / 10000000

		cpupraw := m.Cpu_stats.Online_cpus * (m.Cpu_stats.Cpu_usage.Total_usage - m.Precpu_stats.Cpu_usage.Total_usage) / (cpuTotalUsage - preCpuTotalUsage) / 100000
		cpup = (int)(cpupraw)
		fmt.Println(cpup)
		mempraw := 100.0 * (m.Memory_stats.Usage - m.Memory_stats.Stats.Cache) / m.Memory_stats.Limit
		memp = (int)(mempraw)
		fmt.Println(memp)
	} else {
		cpup = 0
		memp = 0
	}
	csr.Container = name
	// csr.Version = " "
	csr.State = r.State.Status
	csr.CpuRate = cpup
	csr.Memory = memp
	csr.Disk = 0 //未配置
	csr.Ip = r.NetworkSettings.IPAddress
	csr.Created = r.Created
	csr.Started = r.State.StartedAt
	csr.LifeTime = (int64)(lifelong.Hours())
	csr.Image = r.Config.Image

	fmt.Println(csr)
	if err != nil {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = "reportStatus success"
	}
	return code, msg
}
func (gd *GdDocker) reportStatus(param *ServiceReplyParam) (code int, msg string) {

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
		return 400, err.Error()
	}
	paras := []ConStatusReply{}
	pa := ConStatusReply{}
	for _, container := range containers {
		fmt.Printf("reportStatus:%s,%s", container.Names, container.ID)
		//去掉/
		fmt.Println(container.Names[0][1:])
		code, msg = getstatus(cli, ctx, container.Names[0][1:], container.ID, &pa)
		paras = append(paras, pa)
	}

	param.Cmd = REP_CON_STATUS
	param.Paras = paras
	fmt.Println(paras)
	return code, msg
}
func (gd *GdDocker) getIDbyImageName(imagename string) (imageID string, err error) {
	imageID = "_"

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
		return "_", err
	}
	for _, image := range images {
		fmt.Println(image.Containers, image.ID, image.RepoTags, imagename)
		if image.RepoTags[0] == imagename {
			imageID = image.ID
			break
		}
	}
	return imageID, err
}
func (gd *GdDocker) imgRemove(imageName string) (code int, msg string) {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	imageID, err := gd.getIDbyImageName(imageName)
	if err != nil || imageID == "_" {
		code = 400
		if imageID == "_" {
			msg = fmt.Sprintf("container %s don't exit", imageName)
		} else {
			msg = err.Error()
		}
		return code, msg
	}
	fmt.Printf("imgRemove:%s - %s", imageName, imageID)

	imageDeletes, err := cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: false,
	})
	if err != nil || len(imageDeletes) == 0 {
		code = 400
		msg = err.Error()
	} else {
		code = 200
		msg = imageName + " remove success"
	}
	return code, msg
}

// func (gd *GdDocker) imageUpgrade(w http.ResponseWriter, r *http.Request) {
// 	var imageName string
// 	var containerName string
// 	r.ParseForm()
// 	for k, v := range r.Form {
// 		fmt.Println("key:", k)
// 		fmt.Println("val:", strings.Join(v, ""))
// 		if k == "imagename" {
// 			imageName = v[0]
// 			break
// 		}
// 	}

// 	containerName = strings.ReplaceAll(imageName, "/", "-")
// 	containerName = strings.ReplaceAll(containerName, ":", "-")

// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
// 	if err != nil {
// 		panic(err)
// 	}

// 	file, err := os.Open(gd.AtlasImageFullNames[0])
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
// 		Image: imageName,
// 		// Cmd:   []string{"echo", "hello world2"},
// 	}, nil, nil, nil, containerName) //镜像名称作为容器名称
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

// 	fmt.Fprintf(w, "image upgrade success!") //这个写入到w的是输出到客户端的
// }

// func (gd *GdDocker) removeImage(w http.ResponseWriter, r *http.Request) {
// 	var imageID string
// 	var imageName string
// 	r.ParseForm()
// 	for k, v := range r.Form {
// 		fmt.Println("key:", k)
// 		fmt.Println("val:", strings.Join(v, ""))
// 		if k == "imagename" {
// 			imageName = v[0]
// 			break
// 		}
// 	}

// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
// 	if err != nil {
// 		panic(err)
// 	}
// 	imageID = gd.getIDbyImageName(imageName)
// 	imageDeletes, err := cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
// 		Force:         true,
// 		PruneChildren: false,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	if len(imageDeletes) != 2 { //todo
// 		fmt.Printf("expected 2 deleted images, got %v", imageDeletes)
// 	}
// 	fmt.Fprintf(w, "image remove success!") //这个写入到w的是输出到客户端的
// }

func (gd *GdDocker) getIDbyContainerName(containername string) (containerID string, err error) {

	//获取容器名对应的容器id
	containerID = "_"
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
	gd.AtlasImageNames = strings.Fields(config[IMAGENAME])
	gd.AtlasDeviceId = config[DEVICEID]
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

// func (gd *GdDocker) imgUpdate() error {

// 	var containerName string

// 	containerName = strings.ReplaceAll(gd.AtlasImageNames[0], "/", "-")
// 	containerName = strings.ReplaceAll(containerName, ":", "-")

// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
// 	if err != nil {
// 		panic(err)
// 	}
// 	//获取镜像版本，判断是否需要升级
// 	if gd.updateornot() {
// 		file, err := os.Open(gd.AtlasImageFullNames[0])
// 		if err != nil {
// 			fmt.Println("err=", err)
// 		}
// 		imageLoadResponse, err := cli.ImageLoad(ctx, file, true)
// 		if err != nil {
// 			panic(err)
// 		}
// 		body, err := ioutil.ReadAll(imageLoadResponse.Body)
// 		if err != nil {
// 			fmt.Println(" load err=", err)
// 		}
// 		fmt.Println(string(body))

// 		resp, err := cli.ContainerCreate(ctx, &container.Config{
// 			Image: gd.AtlasImageNames[0],
// 			//Cmd:   []string{"echo", "hello world2"},
// 		}, nil, nil, nil, containerName) //镜像名称作为容器名称
// 		if err != nil {
// 			panic(err)
// 		}

// 		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
// 			panic(err)
// 		}

// 		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
// 		select {
// 		case err := <-errCh:
// 			if err != nil {
// 				panic(err)
// 			}
// 		case <-statusCh:
// 		}
// 	} else {
// 		fmt.Println("Current version is updated!")
// 	}

// 	return err
// }
