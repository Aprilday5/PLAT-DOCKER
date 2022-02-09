// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

var ServiceCMDHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPICcmd: %s\n", msg.Topic())
	fmt.Printf("MSGcmd: %s\n", msg.Payload())

	servicecmd := new(ServiceCMD)
	servicereply := new(ServiceReply)
	ca := new(ContainerAPI)
	expectedLogLevel := models.DebugLog
	ca.LoggingClient = logger.NewClient("atlasService", expectedLogLevel)
	//解析cmd，组reply并发布
	err := json.Unmarshal(msg.Payload(), &servicecmd)
	if err != nil {
		panic(err)
	}
	ca.LoggingClient.Debug(servicecmd.Param.Cmd)
	switch servicecmd.Param.Cmd {
	case CMD_CON_INSTALL:
		servicereply = ca.FUNC_CMD_CON_INSTALL(servicecmd)
	// case CMD_STATUS_QUERY:
	// 	ca.FUNC_CMD_STATUS_QUERY()
	case CMD_CON_START:
		servicereply = ca.FUNC_CMD_CON_START(servicecmd)
	case CMD_CON_STOP:
		servicereply = ca.FUNC_CMD_CON_STOP(servicecmd)
	case CMD_CON_REMOVE:
		servicereply = ca.FUNC_CMD_CON_REMOVE(servicecmd)
	// case CMD_CON_SET_CONFIG:
	// 	ca.FUNC_CMD_CON_SET_CONFIG()
	// case CMD_CON_GET_CONFIG:
	// 	ca.FUNC_CMD_CON_GET_CONFIG()
	case CMD_CON_STATUS:
		servicereply = ca.FUNC_CMD_CON_STATUS(servicecmd)
	case CMD_IMG_REMOVE:
		servicereply = ca.FUNC_CMD_IMG_REMOVE(servicecmd)
	// case CMD_CON_UPGRADE:
	// 	ca.FUNC_CMD_CON_UPGRADE()
	// case CMD_CON_LOG:
	// 	ca.FUNC_CMD_CON_LOG()
	default:
		fmt.Printf("there is no such cmd")
	}
	data0, err := json.Marshal(servicereply)
	if err != nil {
		fmt.Println(err)
	}
	token := client.Publish(EDGE_REPLY, 0, false, data0)
	token.Wait()

}
var ServiceReplyHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPICreply: %s\n", msg.Topic())
	fmt.Printf("MSGreply: %s\n", msg.Payload())
}

func main() {
	gd := NewGddocker()
	gd.init()
	gd.loadImage()

	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(gd.EdgeMqttAddress)
	opts.SetClientID("gddockerapp")
	opts.SetDefaultPublishHandler(ServiceDataHandler)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	gd.AtlasDeviceId = getdeviceid(c)
	//subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe(EDGE_CMD, 0, ServiceCMDHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	//Publish 5 messages to /go-mqtt/sample at qos 1 and wait for the receipt
	//from the server after sending each message

	go func() {
		ticker := time.NewTicker(time.Second * 60)
		for range ticker.C {
			//更新data内容
			// servicedata := new(ServiceData)
			// servicedata.Type = "CMD_REPORTDATA"
			// servicedata.Mid = rand.Int63()
			// servicedata.DeviceId = "001"
			// servicedata.Timestamp = time.Now().Unix()
			// servicedata.Param.Cmd = "data" //?
			// servicedata.Param.DeviceId = servicedata.DeviceId
			// servicedata.Param.Data = "datasample"
			ca := new(ContainerAPI)
			expectedLogLevel := models.DebugLog
			ca.LoggingClient = logger.NewClient("atlasService", expectedLogLevel)
			servicedata := ca.FUNC_REP_CON_STATUS()
			data0, err := json.Marshal(servicedata)
			if err != nil {
				fmt.Println(err)
			}

			token := c.Publish(EDGE_DATA, 0, false, data0)
			token.Wait()
		}
	}()
	time.Sleep(3 * time.Second)
	<-waitchan

}

// 主题：/v1/appName/topo/request
// {
// "type":"CMD_TOPO_ADD",
// "mid":1000000000020028,
// "timestamp":1581384683012,
// "expire":-1,
// "param":{
// "nodeInfos":[
// { "nodeId":"atlas200-01",
// "name":"设备之AI加速器",
//  "description":"ATLAS_des",
//  "mfgInfo":"HUAWEI",
//  "nodeModel":"virtal_atlas",
// "modelId":"virtal_atlas" }] } }

// 主题：/v1/appName/topo/response
// {"mid":1000000000020028,
// "deviceId":"2001001000160866",
// "timestamp":1643079340,
// "type":"CMD_TOPO_ADD",
// "code":200,
// "msg":"SUCCESS!",
// "param":{"result":
// [{"statusCode":200,
// "statusDesc":"SUCCESS!",
// "nodeId":"atlas200-01",
// "deviceId":"2001001000160867",
// "profile":{"url":"","name":"","size":0,"md5":""}}]}}
func getdeviceid(c MQTT.Client) (id string) {
	ca := new(ContainerAPI)
	expectedLogLevel := models.DebugLog
	ca.LoggingClient = logger.NewClient("atlasService", expectedLogLevel)
	servicedata := ca.FUNC_REP_CON_STATUS()
	data0, err := json.Marshal(servicedata)
	if err != nil {
		fmt.Println(err)
	}

	token := c.Publish(EDGE_DATA, 0, false, data0)
	token.Wait()
	return id
}
