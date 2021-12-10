// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	MQTT "github.com/eclipse/paho.mqtt.golang"
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
			if strings.Contains(tag, gd.AtlasImageName) { //版本不同
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

	file, err := os.Open(gd.AtlasImageFullName)
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

	containerID = gd.getIDbyContainerName(containerName)

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
func (gd *GdDocker) loadImage(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}

	// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// io.Copy(os.Stdout, reader)
	file, err := os.Open(gd.AtlasImageFullName)
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
func (gd *GdDocker) getIDbyContainerName(containername string) string {
	containerID := "containerid"
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

	containerID = gd.getIDbyContainerName(containerName)
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
func (gd *GdDocker) containerStart(w http.ResponseWriter, r *http.Request) {
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
	containerID = gd.getIDbyContainerName(containerName)
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
func (gd *GdDocker) containerStop(w http.ResponseWriter, r *http.Request) {
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
	containerID = gd.getIDbyContainerName(containerName)
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
	containerID = gd.getIDbyContainerName(containerName)
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

type GdDocker struct {
	AtlasAddress       string
	AtlasImageDir      string
	AtlasImageFullName string
	AtlasImageName     string
}

//读取key=value类型的配置文件
func (gd *GdDocker) InitConfig(path string) {
	config := make(map[string]string)

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		panic(err)
	}

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
	gd.AtlasImageName = config[IMAGENAME]
	fmt.Println(gd)
}
func (gd *GdDocker) init() {
	//读取并获取镜像名称
	//解析文件
	gd.InitConfig(CONF_FILE_PATH)

	//使用文件
	files, _ := ioutil.ReadDir(gd.AtlasImageDir)
	for _, f := range files {
		if strings.Contains(f.Name(), ".tar") {
			gd.AtlasImageFullName = gd.AtlasImageDir + f.Name()
			fmt.Println(gd.AtlasImageFullName)
		}
	}
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

	for _, image := range images {
		fmt.Println(image.Containers, image.ID, image.RepoTags, gd.AtlasImageName)
		if image.RepoTags[0] == gd.AtlasImageName {
			return false
		}
	}
	return true
}
func (gd *GdDocker) imgUpdate() error {

	var containerName string

	containerName = strings.ReplaceAll(gd.AtlasImageName, "/", "-")
	containerName = strings.ReplaceAll(containerName, ":", "-")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(gd.AtlasAddress))
	if err != nil {
		panic(err)
	}
	//获取镜像版本，判断是否需要升级
	if gd.updateornot() {
		file, err := os.Open(gd.AtlasImageFullName)
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
			Image: gd.AtlasImageName,
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

const (
	SERVICE_DATA  = "/v1/appName/service/data"
	SERVICE_CMD   = "/v1/appName/service/command"
	SERVICE_REPLY = "/v1/appName/service/reply"
)

var waitchan = make(chan bool)

type ServiceDataParam struct {
	Cmd      string `json:"cmd"`
	DeviceId string `json:"deviceId"`
	Data     string `json:"data"` //todo,数据以物模型规范的数据格式上报json
}
type ServiceData struct {
	Type      string           `json:"type,omitempty"`
	Mid       int64            `json:"mid,omitempty"`
	DeviceId  string           `json:"deviceId,omitempty"`
	Timestamp int64            `json:"timestamp,omitempty"`
	Expire    int              `json:"expire,omitempty"`
	Param     ServiceDataParam `json:"param,omitempty"`
}

type ServiceCMDParam struct {
	Cmd   string `json:"cmd,omitempty"`
	Paras string `json:"paras,omitempty"` //todo,数据以物模型规范的数据格式上报json
}
type ServiceCMD struct {
	Type      string          `json:"type,omitempty"`
	Mid       int64           `json:"mid,omitempty"`
	DeviceId  string          `json:"deviceId,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	Expire    int             `json:"expire,omitempty"`
	Param     ServiceCMDParam `json:"param,omitempty"`
}

type ServiceRelyParam struct {
	Cmd   string `json:"cmd,omitempty"`
	Paras string `json:"paras,omitempty"` //todo,数据以物模型规范的数据格式上报json
}
type ServiceRely struct {
	Type      string           `json:"type,omitempty"`
	Mid       int64            `json:"mid,omitempty"`
	DeviceId  string           `json:"deviceId,omitempty"`
	Timestamp int64            `json:"timestamp,omitempty"`
	Code      int              `json:"code,omitempty"`
	Msg       string           `json:"msg,omitempty"`
	Param     ServiceRelyParam `json:"param,omitempty"`
}

//define a function for the default message handler
var ServiceDataHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

var ServiceCMDHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPICcmd: %s\n", msg.Topic())
	fmt.Printf("MSGcmd: %s\n", msg.Payload())

	var servicecmd ServiceCMD
	var servicereply ServiceRely
	retcode := 400
	//解析cmd，组reply并发布
	err := json.Unmarshal(msg.Payload(), &servicecmd)
	if err != nil {
		panic(err)
	}
	fmt.Println(servicecmd)

	servicereply.Mid = servicecmd.Mid
	servicereply.Timestamp = time.Now().Unix()
	servicereply.Type = servicecmd.Type
	servicereply.DeviceId = servicecmd.DeviceId
	servicereply.Code = retcode
	servicereply.Msg = "SUCCESS"
	servicereply.Param.Cmd = servicecmd.Param.Cmd
	servicereply.Param.Paras = servicecmd.Param.Paras
	data0, err := json.Marshal(servicereply)
	if err != nil {
		fmt.Println(err)
	}
	token := client.Publish(SERVICE_REPLY, 0, false, data0)
	token.Wait()

}
var ServiceReplyHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPICreply: %s\n", msg.Topic())
	fmt.Printf("MSGreply: %s\n", msg.Payload())
}

func main() {
	gd := new(GdDocker)
	gd.init()
	gd.imgUpdate()

	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://192.168.3.33:1883")
	opts.SetClientID("gddockerapp")
	opts.SetDefaultPublishHandler(ServiceDataHandler)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	//subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe(SERVICE_CMD, 0, ServiceCMDHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	//Publish 5 messages to /go-mqtt/sample at qos 1 and wait for the receipt
	//from the server after sending each message

	go func() {
		ticker := time.NewTicker(time.Second * 2)
		for range ticker.C {
			//更新data内容
			servicedata := new(ServiceData)
			servicedata.Type = "CMD_REPORTDATA"
			servicedata.Mid = rand.Int63()
			servicedata.DeviceId = "001"
			servicedata.Timestamp = time.Now().Unix()
			servicedata.Param.Cmd = "data" //?
			servicedata.Param.DeviceId = servicedata.DeviceId
			servicedata.Param.Data = "datasample"
			data0, err := json.Marshal(servicedata)
			if err != nil {
				fmt.Println(err)
			}

			token := c.Publish(SERVICE_DATA, 0, false, data0)
			token.Wait()
		}
	}()
	time.Sleep(3 * time.Second)
	<-waitchan

}
