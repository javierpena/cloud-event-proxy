//go:build unittests
// +build unittests

package main_test

import (
	"fmt"
	"os"

	"github.com/redhat-cne/cloud-event-proxy/pkg/common"
	"github.com/redhat-cne/sdk-go/pkg/types"

	"sync"
	"testing"

	main "github.com/redhat-cne/cloud-event-proxy/cmd"
	"github.com/redhat-cne/cloud-event-proxy/pkg/plugins"
	"github.com/redhat-cne/sdk-go/pkg/channel"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	v1pubsub "github.com/redhat-cne/sdk-go/v1/pubsub"
)

var (
	channelBufferSize int = 10
	scConfig          *common.SCConfiguration
	resourceAddress   string = "/test/main"
	apiPort           int    = 8990
)

func storeCleanUp() {
	_ = scConfig.PubSubAPI.DeleteAllPublishers()
	_ = scConfig.PubSubAPI.DeleteAllSubscriptions()
}

func TestSidecar_MainWithAMQP(t *testing.T) {
	defer storeCleanUp()
	wg := &sync.WaitGroup{}
	pl := plugins.Handler{Path: "../plugins"}
	// set env variables
	os.Setenv("STORE_PATH", "..")
	var storePath = "."
	if sPath, ok := os.LookupEnv("STORE_PATH"); ok && sPath != "" {
		storePath = sPath
	}
	scConfig = &common.SCConfiguration{
		EventInCh:  make(chan *channel.DataChan, channelBufferSize),
		EventOutCh: make(chan *channel.DataChan, channelBufferSize),
		CloseCh:    make(chan struct{}),
		APIPort:    apiPort,
		APIPath:    "/api/cloudNotifications/v1/",
		PubSubAPI:  v1pubsub.GetAPIInstance(storePath),
		StorePath:  storePath,
		TransportHost: &common.TransportHost{
			Type: common.AMQ,
			URL:  "amqp:localhost:5672",
			Host: "",
			Port: 5672,
			Err:  nil,
		},
	}
	log.Infof("Configuration set to %#v", scConfig)

	//start rest service
	_, err := common.StartPubSubService(scConfig)
	assert.Nil(t, err)

	// imitate main process
	wg.Add(1)
	go main.ProcessOutChannel(wg, scConfig)

	if scConfig.TransportHost.Type == common.AMQ {
		log.Infof("loading amqp with host %s", scConfig.TransportHost.Host)
		_, err = pl.LoadAMQPPlugin(wg, scConfig)
		if err != nil {
			t.Skipf("skipping amqp usage, test will be reading dirctly from in channel. reason: %v", err)
		}
	} else {
		log.Infof("No publishing service enabled")
	}

	//create publisher
	// this is loopback on server itself. Since current pod does not create any server
	endpointURL := fmt.Sprintf("%s%s", scConfig.BaseURL, "dummy")
	createPub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	pub, err := common.CreatePublisher(scConfig, createPub)
	assert.Nil(t, err)
	assert.NotEmpty(t, pub.ID)
	assert.NotEmpty(t, pub.Resource)
	assert.NotEmpty(t, pub.EndPointURI)
	assert.NotEmpty(t, pub.URILocation)
	log.Infof("Publisher \n%s:", pub.String())

	//Test subscription
	createSub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	sub, err := common.CreateSubscription(scConfig, createSub)
	assert.Nil(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.NotEmpty(t, sub.Resource)
	assert.NotEmpty(t, sub.EndPointURI)
	assert.NotEmpty(t, sub.URILocation)
	log.Printf("Subscription \n%s:", sub.String())
	close(scConfig.CloseCh)
}

func TestSidecar_MainWithOutAMQP(t *testing.T) {
	defer storeCleanUp()
	wg := &sync.WaitGroup{}
	// set env variables
	os.Setenv("STORE_PATH", "..")
	var storePath = "."
	if sPath, ok := os.LookupEnv("STORE_PATH"); ok && sPath != "" {
		storePath = sPath
	}
	scConfig = &common.SCConfiguration{
		EventInCh:     make(chan *channel.DataChan, channelBufferSize),
		EventOutCh:    make(chan *channel.DataChan, channelBufferSize),
		CloseCh:       make(chan struct{}),
		APIPort:       apiPort,
		APIPath:       "/api/cloudNotifications/v1/",
		PubSubAPI:     v1pubsub.GetAPIInstance(storePath),
		StorePath:     storePath,
		TransportHost: &common.TransportHost{},
	}
	log.Infof("Configuration set to %#v", scConfig)

	//disable AMQP
	scConfig.PubSubAPI.DisableTransport()

	//start rest service
	_, err := common.StartPubSubService(scConfig)
	assert.Nil(t, err)

	// imitate main process
	wg.Add(1)
	go main.ProcessOutChannel(wg, scConfig)
	wg.Add(1)
	go main.ProcessInChannel(wg, scConfig)

	//create publisher
	// this is loopback on server itself. Since current pod does not create any server
	endpointURL := fmt.Sprintf("%s%s", scConfig.BaseURL, "dummy")
	createPub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	pub, err := common.CreatePublisher(scConfig, createPub)
	assert.Nil(t, err)
	assert.NotEmpty(t, pub.ID)
	assert.NotEmpty(t, pub.Resource)
	assert.NotEmpty(t, pub.EndPointURI)
	assert.NotEmpty(t, pub.URILocation)
	log.Infof("Publisher \n%s:", pub.String())

	//Test subscription
	createSub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	sub, err := common.CreateSubscription(scConfig, createSub)
	assert.Nil(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.NotEmpty(t, sub.Resource)
	assert.NotEmpty(t, sub.EndPointURI)
	assert.NotEmpty(t, sub.URILocation)
	log.Printf("Subscription \n%s:", sub.String())
	close(scConfig.CloseCh)
}

func TestSidecar_MainWithHTTP(t *testing.T) {
	defer storeCleanUp()
	wg := &sync.WaitGroup{}
	pl := plugins.Handler{Path: "../plugins"}
	// set env variables
	os.Setenv("STORE_PATH", "..")
	var storePath = "."
	if sPath, ok := os.LookupEnv("STORE_PATH"); ok && sPath != "" {
		storePath = sPath
	}
	scConfig = &common.SCConfiguration{
		EventInCh:  make(chan *channel.DataChan, channelBufferSize),
		EventOutCh: make(chan *channel.DataChan, channelBufferSize),
		CloseCh:    make(chan struct{}),
		APIPort:    apiPort,
		APIPath:    "/api/cloudNotifications/v1/",
		PubSubAPI:  v1pubsub.GetAPIInstance(storePath),
		StorePath:  storePath,
		TransportHost: &common.TransportHost{
			Type: common.HTTP,
			URL:  "localhost:8089",
			Host: "localhost",
			Port: 8089,
			Err:  nil,
		},
	}
	log.Infof("Configuration set to %#v", scConfig)

	//start rest service
	_, err := common.StartPubSubService(scConfig)
	assert.Nil(t, err)

	// imitate main process
	wg.Add(1)
	go main.ProcessOutChannel(wg, scConfig)

	if scConfig.TransportHost.Type == common.HTTP {
		log.Infof("loading HTTP server with host %s", scConfig.TransportHost.Host)
		_, err = pl.LoadHTTPPlugin(wg, scConfig, nil, nil)
		if err != nil {
			t.Skipf("skipping HTTP usage, test will be reading dirctly from in channel. reason: %v", err)
		}
	} else {
		log.Infof("No publishing service enabled")
	}

	//create publisher
	// this is loopback on server itself. Since current pod does not create any server
	endpointURL := fmt.Sprintf("%s%s", scConfig.BaseURL, "dummy")
	createPub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	pub, err := common.CreatePublisher(scConfig, createPub)
	assert.Nil(t, err)
	assert.NotEmpty(t, pub.ID)
	assert.NotEmpty(t, pub.Resource)
	assert.NotEmpty(t, pub.EndPointURI)
	assert.NotEmpty(t, pub.URILocation)
	log.Infof("Publisher \n%s:", pub.String())

	//Test subscription
	createSub := v1pubsub.NewPubSub(types.ParseURI(endpointURL), resourceAddress)
	sub, err := common.CreateSubscription(scConfig, createSub)
	assert.Nil(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.NotEmpty(t, sub.Resource)
	assert.NotEmpty(t, sub.EndPointURI)
	assert.NotEmpty(t, sub.URILocation)
	log.Printf("Subscription \n%s:", sub.String())
	close(scConfig.CloseCh)
}
